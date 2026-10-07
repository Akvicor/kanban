package service

import (
	"context"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// 卡片字段的长度上限，按字符计算。
const (
	cardTitleMaxLength = 500
	// cardDescriptionMaxLength 是描述的最大字符数，与参考项目一致（1M）。粘贴的图片以 base64 嵌在描述中，因此需要较大的上限。
	cardDescriptionMaxLength = 1048576
)

// copySuffix 返回复制卡片时追加在标题末尾的文字，按用户语言取用，不受支持的语言回落默认语言。
func copySuffix(lang string) string {
	if locale.Type(lang) == locale.En {
		return " (copy)"
	}
	return "(副本)"
}

// Card 是卡片的服务：新建、修改字段、移动、归档、恢复、复制。
var Card = new(cardService)

type cardService struct{}

func validateCardTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", badRequest(resp.CardTitleEmpty, "标题不能为空")
	}
	if utf8.RuneCountInString(title) > cardTitleMaxLength {
		return "", badRequest(resp.CardTitleTooLong, "标题不能超过 500 个字符")
	}
	return title, nil
}

// cardsAndPositions 返回列表中的卡片（按隐藏序号）及其序号，排除 excludeID。
func cardsInList(ctx context.Context, userID, listID int64) ([]*model.Card, error) {
	return repository.Card.ListInList(ctx, userID, listID)
}

// placeCard 计算把卡片放到列表第 index 位（从 0 开始，按隐藏序号，不计它自己）时的序号，index 为 nil 时放到列尾。
// 需要重新排列时修改其他卡片的序号并记录变更。
func placeCard(ctx context.Context, userID, listID, cardID int64, index *int) (int64, error) {
	cards, err := cardsInList(ctx, userID, listID)
	if err != nil {
		return 0, err
	}
	return placeInList(ctx, cards, cardID, index,
		func(c *model.Card) int64 { return c.ID },
		func(c *model.Card) int64 { return *c.Position },
		func(ctx context.Context, c *model.Card, value int64) error {
			c.Position = &value
			if err := repository.Card.Update(ctx, userID, c.ID, map[string]any{"position": value}); err != nil {
				return err
			}
			return recordCard(ctx, c)
		})
}

// endIndex 把列表的一端换算成 placeCard 的位置：列首为 0，列尾为 nil。
func endIndex(end listsort.End) *int {
	if end == listsort.Head {
		return new(int)
	}
	return nil
}

// applyRules 按操作配置计算并写入卡片的标签、开始时间和完成时间。
func applyRules(ctx context.Context, card *model.Card, apply func(listrule.Card) listrule.Card) error {
	labels, err := repository.Card.LabelIDs(ctx, card.UserID, []int64{card.ID})
	if err != nil {
		return err
	}
	result := apply(listrule.Card{Labels: labels[card.ID], StartedAt: card.StartedAt, CompletedAt: card.CompletedAt})
	if !slices.Equal(result.Labels, labels[card.ID]) {
		if err = repository.Card.SetLabels(ctx, card.UserID, card.ID, result.Labels); err != nil {
			return err
		}
	}
	card.StartedAt, card.CompletedAt = result.StartedAt, result.CompletedAt
	return repository.Card.Update(ctx, card.UserID, card.ID, map[string]any{"started_at": card.StartedAt, "completed_at": card.CompletedAt})
}

// rulesOf 读取一个列表的操作配置。
func rulesOf(ctx context.Context, list *model.List) (listrule.Rules, error) {
	rules, err := listRules(ctx, list.UserID, []int64{list.ID})
	if err != nil {
		return listrule.Rules{}, err
	}
	return rules[list.ID], nil
}

// Create 用列首或列尾的创建按钮在列表中新建卡片。卡片插到该按钮配置的一端，并执行该列表的创建配置。
// 创建时间是此刻，之后不变。
func (s *cardService) Create(ctx context.Context, userID, listID int64, title string, button listsort.End) (*model.Card, error) {
	title, err := validateCardTitle(title)
	if err != nil {
		return nil, err
	}
	if !button.Valid() {
		return nil, badRequest(resp.CardButtonInvalid, "创建按钮不正确")
	}
	var card *model.Card
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		list, err := usableList(ctx, userID, listID)
		if err != nil {
			return err
		}
		end := list.TailAdd
		if button == listsort.Head {
			end = list.HeadAdd
		}
		position, err := placeCard(ctx, userID, listID, 0, endIndex(end))
		if err != nil {
			return err
		}
		now := nowUTC()
		card = &model.Card{UserID: userID, PanelID: list.PanelID, ListID: &listID, Position: &position, Title: title, CreatedAt: now}
		if err = repository.Card.Create(ctx, card); err != nil {
			return err
		}
		rules, err := rulesOf(ctx, list)
		if err != nil {
			return err
		}
		if err = applyRules(ctx, card, func(c listrule.Card) listrule.Card { return rules.Create.Apply(c, now) }); err != nil {
			return err
		}
		if err = recordCard(ctx, card); err != nil {
			return err
		}
		return addAction(ctx, card, model.ActionCreate, map[string]any{"list_id": listID}, now)
	})
	return card, err
}

// cardTextField 是做编辑冲突检测的文字字段：列名、记录最后修改序号的列名、冲突提示，
// 以及在卡片模型上读写该字段和最后修改序号的方式。
type cardTextField struct {
	column         string
	revisionColumn string
	conflictMsg    string
	value          func(*model.Card) *string
	changedAt      func(*model.Card) int64
}

var (
	cardTitleField = cardTextField{
		column: "title", revisionColumn: "title_revision", conflictMsg: "标题已在其他设备上修改",
		value:     func(c *model.Card) *string { return &c.Title },
		changedAt: func(c *model.Card) int64 { return c.TitleRevision },
	}
	cardDescriptionField = cardTextField{
		column: "description", revisionColumn: "description_revision", conflictMsg: "描述已在其他设备上修改",
		value:     func(c *model.Card) *string { return &c.Description },
		changedAt: func(c *model.Card) int64 { return c.DescriptionRevision },
	}
)

// UpdateTitle 修改标题。baseRevision 是客户端开始编辑时卡片数据的同步序号；
// 标题在此之后被其他设备修改过时返回带当前卡片的冲突，客户端保留草稿，由用户选择覆盖或放弃。
func (s *cardService) UpdateTitle(ctx context.Context, userID, id int64, title string, baseRevision int64) (*model.Card, error) {
	title, err := validateCardTitle(title)
	if err != nil {
		return nil, err
	}
	return s.updateText(ctx, userID, id, baseRevision, cardTitleField, title)
}

// UpdateDescription 修改描述（Markdown）。冲突规则同 UpdateTitle。
func (s *cardService) UpdateDescription(ctx context.Context, userID, id int64, description string, baseRevision int64) (*model.Card, error) {
	if utf8.RuneCountInString(description) > cardDescriptionMaxLength {
		return nil, badRequest(resp.CardDescriptionTooLong, "描述不能超过 1048576 个字符")
	}
	return s.updateText(ctx, userID, id, baseRevision, cardDescriptionField, description)
}

// updateText 修改标题或描述，并检测编辑冲突：字段的最后修改序号大于 baseRevision 时，
// 返回带当前卡片和读取时同步序号的冲突，不修改任何数据。
func (s *cardService) updateText(ctx context.Context, userID, id, baseRevision int64, field cardTextField, value string) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if card, _, err = usableCard(ctx, userID, id); err != nil {
			return err
		}
		if baseRevision < field.changedAt(card) {
			// 已持有用户锁，读到的卡片与用户当前的同步序号一致。
			user, err := repository.User.FindByID(ctx, userID)
			if err != nil {
				return err
			}
			view, err := cardView(ctx, card)
			if err != nil {
				return err
			}
			return editConflict(field.conflictMsg, view, user.Revision)
		}
		*field.value(card) = value
		if err = repository.Card.Update(ctx, userID, id, map[string]any{field.column: value}); err != nil {
			return err
		}
		if err = recordCard(ctx, card); err != nil {
			return err
		}
		return repository.Card.Update(ctx, userID, id, map[string]any{field.revisionColumn: lastRecordedRevision(ctx, userID)})
	})
	return card, err
}

// SetPriority 设置卡片的优先级挡位，levelID 为 nil 时改为没有优先级。挡位必须属于卡片所在的面板。
func (s *cardService) SetPriority(ctx context.Context, userID, id int64, levelID *int64) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if card, _, err = usableCard(ctx, userID, id); err != nil {
			return err
		}
		if levelID != nil {
			level, err := repository.PriorityLevel.FindByID(ctx, userID, *levelID)
			if err != nil || level.PanelID != card.PanelID {
				return badRequest(resp.CardPriorityOutsidePanel, "优先级挡位不属于卡片所在的面板")
			}
		}
		card.PriorityLevelID = levelID
		if err = repository.Card.Update(ctx, userID, id, map[string]any{"priority_level_id": levelID}); err != nil {
			return err
		}
		return recordCard(ctx, card)
	})
	return card, err
}

// CardDates 是卡片的提醒时间、截止时间和两个通知开关，各项都需要提供，时间为 nil 表示清空。
type CardDates struct {
	RemindAt     *time.Time
	DueAt        *time.Time
	RemindNotify bool
	DueNotify    bool
}

// SetDates 修改提醒时间、截止时间和两个通知开关。
func (s *cardService) SetDates(ctx context.Context, userID, id int64, dates CardDates) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if card, _, err = usableCard(ctx, userID, id); err != nil {
			return err
		}
		truncate := func(t *time.Time) *time.Time {
			if t == nil {
				return nil
			}
			value := t.UTC().Truncate(time.Second)
			return &value
		}
		card.RemindAt, card.DueAt = truncate(dates.RemindAt), truncate(dates.DueAt)
		card.RemindNotify, card.DueNotify = dates.RemindNotify, dates.DueNotify
		err = repository.Card.Update(ctx, userID, id, map[string]any{
			"remind_at": card.RemindAt, "due_at": card.DueAt, "remind_notify": card.RemindNotify, "due_notify": card.DueNotify,
		})
		if err != nil {
			return err
		}
		if err := recordCard(ctx, card); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return card, err
}

// SetLabel 给卡片加上或去掉一个标签。标签必须属于卡片所在的面板。
func (s *cardService) SetLabel(ctx context.Context, userID, id, labelID int64, on bool) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if card, _, err = usableCard(ctx, userID, id); err != nil {
			return err
		}
		label, err := repository.Label.FindByID(ctx, userID, labelID)
		if err != nil || label.PanelID != card.PanelID {
			return badRequest(resp.CardLabelOutsidePanel, "标签不属于卡片所在的面板")
		}
		labels, err := repository.Card.LabelIDs(ctx, userID, []int64{id})
		if err != nil {
			return err
		}
		current := labels[id]
		next := slices.DeleteFunc(slices.Clone(current), func(l int64) bool { return l == labelID })
		if on {
			next = append(next, labelID)
		}
		slices.Sort(next)
		if slices.Equal(next, current) {
			return nil
		}
		if err = repository.Card.SetLabels(ctx, userID, id, next); err != nil {
			return err
		}
		return recordCard(ctx, card)
	})
	return card, err
}

// 定时器操作。
const (
	TimerStart = "start"
	TimerStop  = "stop"
	TimerSet   = "set"
)

// Timer 开始、停止定时器，或直接修改累计秒数。正在计时时修改累计秒数，从此刻起按新值继续计时。
func (s *cardService) Timer(ctx context.Context, userID, id int64, action string, seconds int64) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if card, _, err = usableCard(ctx, userID, id); err != nil {
			return err
		}
		now := nowUTC()
		switch action {
		case TimerStart:
			if card.TimerStartedAt != nil {
				return nil
			}
			card.TimerStartedAt = &now
		case TimerStop:
			return stopTimer(ctx, card, now)
		case TimerSet:
			if seconds < 0 {
				return badRequest(resp.CardElapsedNegative, "累计时间不能为负数")
			}
			card.TimerSeconds = seconds
			if card.TimerStartedAt != nil {
				card.TimerStartedAt = &now
			}
		default:
			return badRequest(resp.CardTimerActionInvalid, "定时器操作不正确")
		}
		err = repository.Card.Update(ctx, userID, id, map[string]any{"timer_seconds": card.TimerSeconds, "timer_started_at": card.TimerStartedAt})
		if err != nil {
			return err
		}
		return recordCard(ctx, card)
	})
	return card, err
}

// Move 把卡片移到同一面板的某个列表的第 index 位（按隐藏序号，从 0 开始），index 为 nil 时放到列尾。
// 同列调整只改序号；跨列移动先执行来源列的移出配置，再执行目标列的移入配置，两步使用同一个时间，并记一条操作。
func (s *cardService) Move(ctx context.Context, userID, id, listID int64, index *int) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var from *model.List
		var err error
		if card, from, err = usableCard(ctx, userID, id); err != nil {
			return err
		}
		to := from
		if listID != from.ID {
			if to, err = usableList(ctx, userID, listID); err != nil {
				return err
			}
			if to.PanelID != card.PanelID {
				return badRequest(resp.CardMoveCrossPanel, "卡片只能在所属面板的列表之间移动")
			}
		}
		position, err := placeCard(ctx, userID, listID, id, index)
		if err != nil {
			return err
		}
		card.ListID, card.Position = &listID, &position
		if err = repository.Card.Update(ctx, userID, id, map[string]any{"list_id": listID, "position": position}); err != nil {
			return err
		}
		if to.ID != from.ID {
			now := nowUTC()
			fromRules, err := rulesOf(ctx, from)
			if err != nil {
				return err
			}
			toRules, err := rulesOf(ctx, to)
			if err != nil {
				return err
			}
			if err = applyRules(ctx, card, func(c listrule.Card) listrule.Card { return listrule.Move(c, &fromRules, &toRules, now) }); err != nil {
				return err
			}
			if err = addAction(ctx, card, model.ActionMove, map[string]any{"from_list_id": from.ID, "to_list_id": to.ID}, now); err != nil {
				return err
			}
		}
		if err = recordCard(ctx, card); err != nil {
			return err
		}
		// 跨列移动会改变所在列表的通知开关是否生效，立即检查。
		if to.ID != from.ID {
			return wakeNotifier(ctx)
		}
		return nil
	})
	return card, err
}

// archiveCard 把卡片放入所属面板的卡片归档：离开列表，停止定时器，记一条操作。不执行移出配置。
func archiveCard(ctx context.Context, card *model.Card, now time.Time) error {
	listID := card.ListID
	if err := stopTimer(ctx, card, now); err != nil {
		return err
	}
	card.ArchivedAt, card.ListID, card.Position = &now, nil, nil
	if err := repository.Card.Update(ctx, card.UserID, card.ID, map[string]any{"archived_at": now, "list_id": nil, "position": nil}); err != nil {
		return err
	}
	if err := recordCard(ctx, card); err != nil {
		return err
	}
	return addAction(ctx, card, model.ActionArchive, map[string]any{"list_id": listID}, now)
}

// Archive 把卡片放入所属面板的卡片归档。
func (s *cardService) Archive(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		card, _, err := usableCard(ctx, userID, id)
		if err != nil {
			return err
		}
		if err = archiveCard(ctx, card, nowUTC()); err != nil {
			return err
		}
		// 归档后卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}

// ArchiveAllInList 把列表中的全部卡片逐张放入卡片归档，在同一个事务里完成。列表本身保留。
func (s *cardService) ArchiveAllInList(ctx context.Context, userID, listID int64) error {
	return write(ctx, func(ctx context.Context) error {
		if _, err := usableList(ctx, userID, listID); err != nil {
			return err
		}
		cards, err := cardsInList(ctx, userID, listID)
		if err != nil {
			return err
		}
		now := nowUTC()
		for _, card := range cards {
			if err = archiveCard(ctx, card, now); err != nil {
				return err
			}
		}
		// 归档后卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}

// Restore 把卡片归档中的卡片恢复到该面板的某个列表的列首或列尾。只改所在列和序号，
// 不执行目标列的移入配置，不重新启动定时器；记一条操作，数据是目标列和插入端。
func (s *cardService) Restore(ctx context.Context, userID, id, listID int64, end listsort.End) (*model.Card, error) {
	if !end.Valid() {
		return nil, badRequest(resp.PositionInvalid, "插入位置不正确")
	}
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if card, err = findCard(ctx, userID, id); err != nil {
			return err
		}
		if !card.Archived() {
			return badRequest(resp.CardNotArchived, "卡片不在卡片归档中")
		}
		list, err := usableList(ctx, userID, listID)
		if err != nil {
			return err
		}
		if list.PanelID != card.PanelID {
			return badRequest(resp.CardRestoreWrongPanel, "只能恢复到卡片所属面板的列表")
		}
		position, err := placeCard(ctx, userID, listID, id, endIndex(end))
		if err != nil {
			return err
		}
		card.ArchivedAt, card.ListID, card.Position = nil, &listID, &position
		if err = repository.Card.Update(ctx, userID, id, map[string]any{"archived_at": nil, "list_id": listID, "position": position}); err != nil {
			return err
		}
		if err = addAction(ctx, card, model.ActionRestore, map[string]any{"list_id": listID, "end": end}, nowUTC()); err != nil {
			return err
		}
		if err := recordCardBundle(ctx, []*model.Card{card}); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return card, err
}

// Copy 在所属面板内复制卡片。targetListID 为 nil 时副本插在原卡片下方（菜单中的复制）；
// 否则插到目标列表的列首（快捷键粘贴）。两种情况都执行目标列表的创建配置。
//
// 副本带上标题（末尾加「(副本)」）、描述、标签、优先级、任务（保留层级和完成状态）、提醒和截止时间及通知开关、关联、
// 附件（引用同一份文件，用户引用数增加）和封面（指向副本中对应的附件）；
// 操作记录、定时器和创建、开始、完成时间不复制。副本记一条创建操作，数据中注明复制来源。
func (s *cardService) Copy(ctx context.Context, userID, id int64, targetListID *int64) (*model.Card, error) {
	user, err := User.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	var copied *model.Card
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		source, sourceList, err := usableCard(ctx, userID, id)
		if err != nil {
			return err
		}
		list := sourceList
		var index *int
		if targetListID == nil {
			cards, err := cardsInList(ctx, userID, sourceList.ID)
			if err != nil {
				return err
			}
			below := slices.IndexFunc(cards, func(c *model.Card) bool { return c.ID == id }) + 1
			index = &below
		} else {
			if list, err = usableList(ctx, userID, *targetListID); err != nil {
				return err
			}
			if list.PanelID != source.PanelID {
				return badRequest(resp.CardCopyCrossPanel, "卡片只能在所属面板内复制")
			}
			index = new(int)
		}
		position, err := placeCard(ctx, userID, list.ID, 0, index)
		if err != nil {
			return err
		}

		now := nowUTC()
		listID := list.ID
		copied = &model.Card{
			UserID: userID, PanelID: source.PanelID, ListID: &listID, Position: &position,
			Title: source.Title + copySuffix(uiLang(ctx, user)), Description: source.Description, PriorityLevelID: source.PriorityLevelID,
			RemindAt: source.RemindAt, DueAt: source.DueAt, RemindNotify: source.RemindNotify, DueNotify: source.DueNotify,
			CreatedAt: now,
		}
		if err = repository.Card.Create(ctx, copied); err != nil {
			return err
		}
		labels, err := repository.Card.LabelIDs(ctx, userID, []int64{id})
		if err != nil {
			return err
		}
		if err = repository.Card.SetLabels(ctx, userID, copied.ID, labels[id]); err != nil {
			return err
		}
		if err = copyTasks(ctx, userID, id, copied.ID); err != nil {
			return err
		}
		if err = copyLinks(ctx, userID, id, copied.ID); err != nil {
			return err
		}
		if err = copyCardChannels(ctx, userID, id, copied.ID); err != nil {
			return err
		}
		attachments, err := copyAttachments(ctx, userID, id, copied.ID, now)
		if err != nil {
			return err
		}
		if source.CoverAttachmentID != nil {
			if cover, ok := attachments[*source.CoverAttachmentID]; ok {
				copied.CoverAttachmentID = &cover
				if err = repository.Card.Update(ctx, userID, copied.ID, map[string]any{"cover_attachment_id": cover}); err != nil {
					return err
				}
			}
		}
		rules, err := rulesOf(ctx, list)
		if err != nil {
			return err
		}
		if err = applyRules(ctx, copied, func(c listrule.Card) listrule.Card { return rules.Create.Apply(c, now) }); err != nil {
			return err
		}
		if err = recordCard(ctx, copied); err != nil {
			return err
		}
		if err := addAction(ctx, copied, model.ActionCreate, map[string]any{"list_id": list.ID, "copied_from": id}, now); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return copied, err
}

// copyTasks 把原卡片的任务按原有层级和完成状态复制到副本。
func copyTasks(ctx context.Context, userID, fromCardID, toCardID int64) error {
	tasks, err := repository.Task.ListByCards(ctx, userID, []int64{fromCardID})
	if err != nil {
		return err
	}
	// 按层级从上到下复制，保证父任务先于子任务创建。
	mapping := map[int64]int64{}
	pending := tasks
	for len(pending) > 0 {
		var next []*model.Task
		for _, task := range pending {
			var parentID *int64
			if task.ParentID != nil {
				newParent, ok := mapping[*task.ParentID]
				if !ok {
					next = append(next, task)
					continue
				}
				parentID = &newParent
			}
			copied := &model.Task{UserID: userID, CardID: toCardID, ParentID: parentID, Title: task.Title, Done: task.Done, Position: task.Position}
			if err = repository.Task.Create(ctx, copied); err != nil {
				return err
			}
			if err = recordTask(ctx, copied); err != nil {
				return err
			}
			mapping[task.ID] = copied.ID
		}
		if len(next) == len(pending) {
			break // 父任务缺失的孤立任务不复制
		}
		pending = next
	}
	return nil
}

// copyLinks 把原卡片的关联按原顺序复制到副本。
func copyLinks(ctx context.Context, userID, fromCardID, toCardID int64) error {
	links, err := repository.CardLink.ListByCards(ctx, userID, []int64{fromCardID})
	if err != nil {
		return err
	}
	for _, link := range links {
		copied := &model.CardLink{UserID: userID, CardID: toCardID, BoardID: link.BoardID, PanelID: link.PanelID, Position: link.Position}
		if err = repository.CardLink.Create(ctx, copied); err != nil {
			return err
		}
		if err = recordCardLink(ctx, copied); err != nil {
			return err
		}
	}
	return nil
}

// ArchivedInPanel 返回面板卡片归档中的卡片，连同任务、操作记录和关联，以及读取时的同步序号。
// 只有面板正常使用时才能进入卡片归档。
func (s *cardService) ArchivedInPanel(ctx context.Context, userID, panelID int64) (*dro.CardBundle, error) {
	return readBundle(ctx, userID, func(ctx context.Context) ([]*model.Card, error) {
		if _, _, err := usablePanel(ctx, userID, panelID); err != nil {
			return nil, err
		}
		return repository.Card.ListArchivedInPanel(ctx, userID, panelID)
	})
}

// InArchivedList 返回列表归档中某个列表的卡片，用于只读查看。只有面板正常使用时才能进入列表归档。
func (s *cardService) InArchivedList(ctx context.Context, userID, listID int64) (*dro.CardBundle, error) {
	return readBundle(ctx, userID, func(ctx context.Context) ([]*model.Card, error) {
		list, err := findList(ctx, userID, listID)
		if err != nil {
			return nil, err
		}
		if _, _, err = usablePanel(ctx, userID, list.PanelID); err != nil {
			return nil, err
		}
		return cardsInList(ctx, userID, listID)
	})
}

// readBundle 在只读事务中读取卡片和同步序号。
func readBundle(ctx context.Context, userID int64, load func(ctx context.Context) ([]*model.Card, error)) (*dro.CardBundle, error) {
	var bundle *dro.CardBundle
	err := repository.ReadTransaction(ctx, func(ctx context.Context) error {
		user, err := User.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		cards, err := load(ctx)
		if err != nil {
			return err
		}
		if bundle, err = cardBundle(ctx, userID, cards); err != nil {
			return err
		}
		bundle.Revision = user.Revision
		return nil
	})
	return bundle, err
}

// View 返回带标签的卡片信息，用于写入后的响应。
func (s *cardService) View(ctx context.Context, card *model.Card) (dro.Card, error) {
	return cardView(ctx, card)
}
