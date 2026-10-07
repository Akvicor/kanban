package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"slices"
	"time"

	"gorm.io/gorm"
)

// EntityList 是列表的同步实体类型。列表的数据包含它的操作配置。
const EntityList = "list"

// List 是列表的服务：新建、改名、设置、操作配置、排序、归档、恢复，以及打开面板时加载列表。
var List = new(listService)

type listService struct{}

// ListSettings 是列表的展示和通知设置。Color 为空表示没有颜色。
type ListSettings struct {
	Color     string
	ShowAge   bool
	SortMode  listsort.Mode
	SortDir   listsort.Direction
	HeadAdd   listsort.End
	TailAdd   listsort.End
	RemindOff bool
	DueOff    bool
}

// validate 校验设置取值，颜色统一为大写。
func (s *ListSettings) validate() error {
	if s.Color != "" {
		color, err := validateColor(s.Color)
		if err != nil {
			return err
		}
		s.Color = color
	}
	if !s.SortMode.Valid() || !s.SortDir.Valid() {
		return badRequest(resp.ListSortInvalid, "排序方式不正确")
	}
	if !s.HeadAdd.Valid() || !s.TailAdd.Valid() {
		return badRequest(resp.PositionInvalid, "插入位置不正确")
	}
	return nil
}

func findList(ctx context.Context, userID, id int64) (*model.List, error) {
	list, err := repository.List.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.ListNotFound, "列表不存在")
	}
	return list, err
}

// usableList 查找可以修改的列表：列表不在列表归档中，所在面板正常使用中。
func usableList(ctx context.Context, userID, id int64) (*model.List, error) {
	list, err := findList(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if list.Archived() {
		return nil, badRequest(resp.ListArchived, "列表已归档")
	}
	if _, _, err = usablePanel(ctx, userID, list.PanelID); err != nil {
		return nil, err
	}
	return list, nil
}

// listRules 读取这些列表的操作配置，按列表 ID 归组。
func listRules(ctx context.Context, userID int64, listIDs []int64) (map[int64]listrule.Rules, error) {
	result := make(map[int64]listrule.Rules, len(listIDs))
	if len(listIDs) == 0 {
		return result, nil
	}
	labelRules, err := repository.List.LabelRules(ctx, userID, listIDs)
	if err != nil {
		return nil, err
	}
	timeRules, err := repository.List.TimeRules(ctx, userID, listIDs)
	if err != nil {
		return nil, err
	}
	for _, rule := range labelRules {
		rules := result[rule.ListID]
		op := rules.For(rule.Op)
		if rule.Action == model.LabelAdd {
			op.AddLabels = append(op.AddLabels, rule.LabelID)
		} else {
			op.RemoveLabels = append(op.RemoveLabels, rule.LabelID)
		}
		result[rule.ListID] = rules
	}
	for _, rule := range timeRules {
		rules := result[rule.ListID]
		op := rules.For(rule.Op)
		if rule.Field == model.FieldStart {
			op.Start = rule.Action
		} else {
			op.Complete = rule.Action
		}
		result[rule.ListID] = rules
	}
	return result, nil
}

// normalizeRules 把空的标签列表统一为空数组，便于客户端处理。
func normalizeRules(rules listrule.Rules) listrule.Rules {
	for _, op := range listrule.Ops {
		r := rules.For(op)
		if r.AddLabels == nil {
			r.AddLabels = []int64{}
		}
		if r.RemoveLabels == nil {
			r.RemoveLabels = []int64{}
		}
	}
	return rules
}

// listView 生成带操作配置的列表信息。
func listView(ctx context.Context, list *model.List) (dro.List, error) {
	rules, err := listRules(ctx, list.UserID, []int64{list.ID})
	if err != nil {
		return dro.List{}, err
	}
	return dro.NewList(list, normalizeRules(rules[list.ID])), nil
}

func recordList(ctx context.Context, list *model.List) error {
	view, err := listView(ctx, list)
	if err != nil {
		return err
	}
	return recordChange(ctx, list.UserID, hub.OpUpsert, EntityList, list.ID, view)
}

// Content 返回面板的全部列表（包括列表归档中的）及其操作配置、其中未归档列表里的卡片连同下级内容，以及读取时的同步序号。
// 面板在归档中时也可以加载，用于只读查看。
func (s *listService) Content(ctx context.Context, userID, panelID int64) (*dro.PanelContent, error) {
	content := &dro.PanelContent{PanelID: panelID}
	err := repository.ReadTransaction(ctx, func(ctx context.Context) error {
		user, err := User.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		if _, err = findPanel(ctx, userID, panelID); err != nil {
			return err
		}
		lists, err := repository.List.ListByPanel(ctx, userID, panelID)
		if err != nil {
			return err
		}
		ids := make([]int64, len(lists))
		for i, list := range lists {
			ids[i] = list.ID
		}
		rules, err := listRules(ctx, userID, ids)
		if err != nil {
			return err
		}
		content.Lists = make([]dro.List, 0, len(lists))
		activeIDs := make([]int64, 0, len(lists))
		for _, list := range lists {
			content.Lists = append(content.Lists, dro.NewList(list, normalizeRules(rules[list.ID])))
			if !list.Archived() {
				activeIDs = append(activeIDs, list.ID)
			}
		}
		cards, err := repository.Card.ListInLists(ctx, userID, activeIDs)
		if err != nil {
			return err
		}
		bundle, err := cardBundle(ctx, userID, cards)
		if err != nil {
			return err
		}
		content.CardBundle = *bundle
		content.Revision = user.Revision
		return nil
	})
	if err != nil {
		return nil, err
	}
	return content, nil
}

// View 返回列表连同操作配置的信息，用于写入后的响应。
func (s *listService) View(ctx context.Context, list *model.List) (dro.List, error) {
	return listView(ctx, list)
}

// activeListPositions 返回面板中未归档列表的排序值，按从左到右。
func activeListPositions(ctx context.Context, userID, panelID int64) ([]*model.List, []int64, error) {
	lists, err := repository.List.ListActiveByPanel(ctx, userID, panelID)
	if err != nil {
		return nil, nil, err
	}
	positions := make([]int64, len(lists))
	for i, list := range lists {
		positions[i] = list.Position
	}
	return lists, positions, nil
}

// Create 在面板最右边新建列表。新列表使用默认设置：显示卡龄、手动排序、列首按钮插到列首、列尾按钮插到列尾、
// 不关闭通知、没有操作配置。
func (s *listService) Create(ctx context.Context, userID, panelID int64, name string) (*model.List, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	list := &model.List{
		UserID: userID, PanelID: panelID, Name: name,
		ShowAge: true, SortMode: listsort.Manual, SortDir: listsort.Asc, HeadAdd: listsort.Head, TailAdd: listsort.Tail,
		CreatedAt: time.Now().UTC(),
	}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, _, err := usablePanel(ctx, userID, panelID); err != nil {
			return err
		}
		_, positions, err := activeListPositions(ctx, userID, panelID)
		if err != nil {
			return err
		}
		list.Position, _ = slotPosition(positions, nil)
		if err = repository.List.Create(ctx, list); err != nil {
			return err
		}
		return recordList(ctx, list)
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Rename 修改列表名称。
func (s *listService) Rename(ctx context.Context, userID, id int64, name string) (*model.List, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	var list *model.List
	err = write(ctx, func(ctx context.Context) error {
		if list, err = usableList(ctx, userID, id); err != nil {
			return err
		}
		list.Name = name
		if err = repository.List.Update(ctx, userID, id, map[string]any{"name": name}); err != nil {
			return err
		}
		return recordList(ctx, list)
	})
	return list, err
}

// UpdateSettings 修改列表的展示和通知设置。通知开关只影响当时位于该列的卡片，不修改卡片自己的时间和通知开关。
func (s *listService) UpdateSettings(ctx context.Context, userID, id int64, settings ListSettings) (*model.List, error) {
	if err := settings.validate(); err != nil {
		return nil, err
	}
	var list *model.List
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if list, err = usableList(ctx, userID, id); err != nil {
			return err
		}
		list.Color, list.ShowAge = settings.Color, settings.ShowAge
		list.SortMode, list.SortDir = settings.SortMode, settings.SortDir
		list.HeadAdd, list.TailAdd = settings.HeadAdd, settings.TailAdd
		list.RemindOff, list.DueOff = settings.RemindOff, settings.DueOff
		err = repository.List.Update(ctx, userID, id, map[string]any{
			"color": list.Color, "show_age": list.ShowAge, "sort_mode": list.SortMode, "sort_dir": list.SortDir,
			"head_add": list.HeadAdd, "tail_add": list.TailAdd, "remind_off": list.RemindOff, "due_off": list.DueOff,
		})
		if err != nil {
			return err
		}
		if err := recordList(ctx, list); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return list, err
}

// UpdateRules 用新的操作配置替换列表原有的配置。标签必须属于列表所在的面板；同一操作中重复的标签只保留一条。
func (s *listService) UpdateRules(ctx context.Context, userID, id int64, rules listrule.Rules) (*model.List, error) {
	var list *model.List
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if list, err = usableList(ctx, userID, id); err != nil {
			return err
		}
		labels, err := repository.Label.ListByPanel(ctx, userID, list.PanelID)
		if err != nil {
			return err
		}
		panelLabels := make([]int64, len(labels))
		for i, label := range labels {
			panelLabels[i] = label.ID
		}

		var labelRules []*model.ListLabelRule
		var timeRules []*model.ListTimeRule
		for _, op := range listrule.Ops {
			r := rules.For(op)
			for action, ids := range map[string][]int64{model.LabelAdd: r.AddLabels, model.LabelRemove: r.RemoveLabels} {
				seen := map[int64]bool{}
				for _, labelID := range ids {
					if !slices.Contains(panelLabels, labelID) {
						return badRequest(resp.ListRuleLabelOutside, "操作配置只能使用本面板的标签")
					}
					if seen[labelID] {
						continue
					}
					seen[labelID] = true
					labelRules = append(labelRules, &model.ListLabelRule{UserID: userID, ListID: id, Op: op, Action: action, LabelID: labelID})
				}
			}
			for field, action := range map[string]listrule.TimeAction{model.FieldStart: r.Start, model.FieldComplete: r.Complete} {
				if !action.Valid() {
					return badRequest(resp.ListTimeActionInvalid, "时间动作不正确")
				}
				if action != listrule.TimeNone {
					timeRules = append(timeRules, &model.ListTimeRule{UserID: userID, ListID: id, Op: op, Field: field, Action: action})
				}
			}
		}
		if err = repository.List.ReplaceRules(ctx, userID, id, labelRules, timeRules); err != nil {
			return err
		}
		return recordList(ctx, list)
	})
	return list, err
}

// Reorder 把列表移到面板中从左数第 index 位，index 为 nil 时放到最右边。
func (s *listService) Reorder(ctx context.Context, userID, id int64, index *int) (*model.List, error) {
	var list *model.List
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if list, err = usableList(ctx, userID, id); err != nil {
			return err
		}
		lists, _, err := activeListPositions(ctx, userID, list.PanelID)
		if err != nil {
			return err
		}
		save := func(ctx context.Context, l *model.List, value int64) error {
			l.Position = value
			if err := repository.List.Update(ctx, userID, l.ID, map[string]any{"position": value}); err != nil {
				return err
			}
			return recordList(ctx, l)
		}
		position, err := placeInList(ctx, lists, id, index,
			func(l *model.List) int64 { return l.ID },
			func(l *model.List) int64 { return l.Position },
			save)
		if err != nil {
			return err
		}
		return save(ctx, list, position)
	})
	return list, err
}

// Archive 把列表连同其中的卡片放入所属面板的列表归档。卡片的所在列和隐藏序号保持不变，不执行移出配置；
// 其中正在计时的定时器停止。
func (s *listService) Archive(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		list, err := usableList(ctx, userID, id)
		if err != nil {
			return err
		}
		now := nowUTC()
		cards, err := cardsInList(ctx, userID, id)
		if err != nil {
			return err
		}
		for _, card := range cards {
			if err = stopTimer(ctx, card, now); err != nil {
				return err
			}
		}
		list.ArchivedAt = &now
		if err = repository.List.Update(ctx, userID, id, map[string]any{"archived_at": now}); err != nil {
			return err
		}
		if err = recordList(ctx, list); err != nil {
			return err
		}
		// 归档后其中的卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}

// Restore 把列表归档中的列表连同卡片、配置和卡片顺序恢复到面板的最左边（atStart）或最右边。
// 只有面板正常使用时才能恢复。
func (s *listService) Restore(ctx context.Context, userID, id int64, atStart bool) (*model.List, error) {
	var list *model.List
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if list, err = findList(ctx, userID, id); err != nil {
			return err
		}
		if !list.Archived() {
			return badRequest(resp.ListNotArchived, "列表不在列表归档中")
		}
		if _, _, err = usablePanel(ctx, userID, list.PanelID); err != nil {
			return err
		}
		_, positions, err := activeListPositions(ctx, userID, list.PanelID)
		if err != nil {
			return err
		}
		var index *int
		if atStart {
			index = new(int)
		}
		// 恢复到最左边时，最左边的排序值减去一个间隙即可，不会触发重排。
		list.Position, _ = slotPosition(positions, index)
		list.ArchivedAt = nil
		if err = repository.List.Update(ctx, userID, id, map[string]any{"archived_at": nil, "position": list.Position}); err != nil {
			return err
		}
		if err = recordList(ctx, list); err != nil {
			return err
		}
		// 打开面板时只加载不在列表归档中的列表里的卡片，因此把恢复的列表中的卡片完整推送一次。
		cards, err := cardsInList(ctx, userID, id)
		if err != nil {
			return err
		}
		if err := recordCardBundle(ctx, cards); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return list, err
}
