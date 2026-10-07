package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"time"

	"gorm.io/gorm"
)

// 同步的实体类型。
const (
	EntityPanel         = "panel"
	EntityLabel         = "label"
	EntityPriorityLevel = "priority_level"
)

// defaultPanelName 是新建看板时自动创建的面板名称，按用户语言取用，不受支持的语言回落默认语言。
func defaultPanelName(lang string) string {
	if locale.Type(lang) == locale.En {
		return "Default"
	}
	return "默认"
}

// defaultPriorityLevels 是新建面板的默认优先级挡位，按优先级从高到低排列。
var defaultPriorityLevels = []struct{ name, color string }{
	{"P0", "#E5484D"},
	{"P1", "#F76B15"},
	{"P2", "#3E63DD"},
	{"P3", "#8B8D98"},
}

// Panel 是面板的服务：新建、改名、排序、移到其他看板、归档、恢复。
var Panel = new(panelService)

type panelService struct{}

// usablePanel 查找正常使用中的面板：面板不在面板归档中，所在看板也不在看板归档中。返回面板和所在看板。
func usablePanel(ctx context.Context, userID, id int64) (*model.Panel, *model.Board, error) {
	panel, err := findPanel(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	if panel.Archived() {
		return nil, nil, badRequest(resp.PanelArchived, "面板已归档")
	}
	board, err := findBoard(ctx, userID, panel.BoardID)
	if err != nil {
		return nil, nil, err
	}
	if board.Archived() {
		return nil, nil, badRequest(resp.PanelBoardArchived, "面板所在的看板已归档")
	}
	return panel, board, nil
}

func findPanel(ctx context.Context, userID, id int64) (*model.Panel, error) {
	panel, err := repository.Panel.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.PanelNotFound, "面板不存在")
	}
	return panel, err
}

func recordPanel(ctx context.Context, panel *model.Panel) error {
	return recordChange(ctx, panel.UserID, hub.OpUpsert, EntityPanel, panel.ID, dro.NewPanel(panel))
}

// endOfBoard 返回把面板放到看板最后一个标签页时的排序值。
func endOfBoard(ctx context.Context, userID, boardID int64) (int64, error) {
	panels, err := repository.Panel.ListActiveInBoard(ctx, userID, boardID)
	if err != nil {
		return 0, err
	}
	positions := make([]int64, len(panels))
	for i, panel := range panels {
		positions[i] = panel.Position
	}
	position, _ := slotPosition(positions, nil)
	return position, nil
}

// createPanel 在看板最后新建面板，并创建默认优先级挡位。必须在 write 中调用。
func createPanel(ctx context.Context, board *model.Board, name string) (*model.Panel, error) {
	position, err := endOfBoard(ctx, board.UserID, board.ID)
	if err != nil {
		return nil, err
	}
	panel := &model.Panel{UserID: board.UserID, BoardID: board.ID, Name: name, Position: position, CreatedAt: time.Now().UTC()}
	if err = repository.Panel.Create(ctx, panel); err != nil {
		return nil, err
	}
	if err = recordPanel(ctx, panel); err != nil {
		return nil, err
	}
	for i, level := range defaultPriorityLevels {
		priority := &model.PriorityLevel{
			UserID: board.UserID, PanelID: panel.ID, Name: level.name, Color: level.color, Position: int64(i+1) * positionGap,
		}
		if err = repository.PriorityLevel.Create(ctx, priority); err != nil {
			return nil, err
		}
		if err = recordPriorityLevel(ctx, priority); err != nil {
			return nil, err
		}
	}
	return panel, nil
}

// clearMainPanel 在看板的主面板是 panelID 时清空主面板指向。
func clearMainPanel(ctx context.Context, userID, boardID, panelID int64) error {
	board, err := findBoard(ctx, userID, boardID)
	if err != nil {
		return err
	}
	if board.MainPanelID == nil || *board.MainPanelID != panelID {
		return nil
	}
	board.MainPanelID = nil
	if err = repository.Board.Update(ctx, userID, boardID, map[string]any{"main_panel_id": nil}); err != nil {
		return err
	}
	return recordBoard(ctx, board)
}

// Create 在看板最后新建面板。
func (s *panelService) Create(ctx context.Context, userID, boardID int64, name string) (*model.Panel, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	var panel *model.Panel
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		board, err := activeBoard(ctx, userID, boardID)
		if err != nil {
			return err
		}
		panel, err = createPanel(ctx, board, name)
		return err
	})
	return panel, err
}

// Rename 修改面板名称。
func (s *panelService) Rename(ctx context.Context, userID, id int64, name string) (*model.Panel, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	var panel *model.Panel
	err = write(ctx, func(ctx context.Context) error {
		if panel, _, err = usablePanel(ctx, userID, id); err != nil {
			return err
		}
		panel.Name = name
		if err = repository.Panel.Update(ctx, userID, id, map[string]any{"name": name}); err != nil {
			return err
		}
		return recordPanel(ctx, panel)
	})
	return panel, err
}

// Reorder 把面板移到所在看板标签页的第 index 位，index 为 nil 时放到最后。
func (s *panelService) Reorder(ctx context.Context, userID, id int64, index *int) (*model.Panel, error) {
	var panel *model.Panel
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if panel, _, err = usablePanel(ctx, userID, id); err != nil {
			return err
		}
		siblings, err := repository.Panel.ListActiveInBoard(ctx, userID, panel.BoardID)
		if err != nil {
			return err
		}
		position, err := placeInList(ctx, siblings, id, index,
			func(p *model.Panel) int64 { return p.ID },
			func(p *model.Panel) int64 { return p.Position },
			func(ctx context.Context, p *model.Panel, value int64) error {
				p.Position = value
				if err := repository.Panel.Update(ctx, userID, p.ID, map[string]any{"position": value}); err != nil {
					return err
				}
				return recordPanel(ctx, p)
			})
		if err != nil {
			return err
		}
		panel.Position = position
		if err = repository.Panel.Update(ctx, userID, id, map[string]any{"position": position}); err != nil {
			return err
		}
		return recordPanel(ctx, panel)
	})
	return panel, err
}

// MoveToBoard 把面板连同其中的全部内容移到另一个未归档的看板，排在最后一个标签页。
// 面板原来是原看板的主面板时清空原看板的主面板指向；面板不会成为目标看板的主面板。
func (s *panelService) MoveToBoard(ctx context.Context, userID, id, targetBoardID int64) (*model.Panel, error) {
	var panel *model.Panel
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if panel, _, err = usablePanel(ctx, userID, id); err != nil {
			return err
		}
		if panel.BoardID == targetBoardID {
			return badRequest(resp.PanelAlreadyInBoard, "面板已经在这个看板中")
		}
		if _, err = activeBoard(ctx, userID, targetBoardID); err != nil {
			return err
		}
		return relocatePanel(ctx, panel, targetBoardID)
	})
	return panel, err
}

// Archive 把面板放入面板归档。面板是主面板时清空所在看板的主面板指向；其中正在计时的定时器停止。
func (s *panelService) Archive(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		panel, _, err := usablePanel(ctx, userID, id)
		if err != nil {
			return err
		}
		now := nowUTC()
		if err = stopTimersInPanels(ctx, userID, []int64{id}, now); err != nil {
			return err
		}
		panel.ArchivedAt = &now
		if err = repository.Panel.Update(ctx, userID, id, map[string]any{"archived_at": now}); err != nil {
			return err
		}
		if err = recordPanel(ctx, panel); err != nil {
			return err
		}
		if err = clearMainPanel(ctx, userID, panel.BoardID, id); err != nil {
			return err
		}
		// 归档后其中的卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}

// Restore 把归档中的面板恢复到一个未归档的看板，排在最后一个标签页。面板可以来自两处：
//   - 面板归档：恢复后清空归档时间。
//   - 看板归档中某个看板里的面板：面板从该看板移出，不再随该看板恢复。
//
// 两种情况都不会让面板成为目标看板的主面板，恢复回原看板时也一样。
func (s *panelService) Restore(ctx context.Context, userID, id, targetBoardID int64) (*model.Panel, error) {
	var panel *model.Panel
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if panel, err = findPanel(ctx, userID, id); err != nil {
			return err
		}
		if !panel.Archived() {
			board, err := findBoard(ctx, userID, panel.BoardID)
			if err != nil {
				return err
			}
			if !board.Archived() {
				return badRequest(resp.PanelNotArchived, "面板不在归档中")
			}
		}
		if _, err = activeBoard(ctx, userID, targetBoardID); err != nil {
			return err
		}
		panel.ArchivedAt = nil
		if err := relocatePanel(ctx, panel, targetBoardID); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return panel, err
}

// relocatePanel 把面板放到目标看板的最后，连同面板当前的归档时间一起写入，并清空原看板对它的主面板指向。
func relocatePanel(ctx context.Context, panel *model.Panel, targetBoardID int64) error {
	originBoardID := panel.BoardID
	position, err := endOfBoard(ctx, panel.UserID, targetBoardID)
	if err != nil {
		return err
	}
	panel.BoardID, panel.Position = targetBoardID, position
	err = repository.Panel.Update(ctx, panel.UserID, panel.ID, map[string]any{
		"board_id": targetBoardID, "position": position, "archived_at": panel.ArchivedAt,
	})
	if err != nil {
		return err
	}
	if err = recordPanel(ctx, panel); err != nil {
		return err
	}
	if originBoardID == targetBoardID {
		return nil
	}
	return clearMainPanel(ctx, panel.UserID, originBoardID, panel.ID)
}

// SetMain 设置看板的主面板；panelID 为 nil 时取消。主面板必须属于该看板且未归档。
func (s *panelService) SetMain(ctx context.Context, userID, boardID int64, panelID *int64) (*model.Board, error) {
	var board *model.Board
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if board, err = activeBoard(ctx, userID, boardID); err != nil {
			return err
		}
		if panelID != nil {
			panel, _, err := usablePanel(ctx, userID, *panelID)
			if err != nil {
				return err
			}
			if panel.BoardID != boardID {
				return badRequest(resp.PanelWrongBoard, "面板不属于这个看板")
			}
		}
		board.MainPanelID = panelID
		if err = repository.Board.Update(ctx, userID, boardID, map[string]any{"main_panel_id": panelID}); err != nil {
			return err
		}
		return recordBoard(ctx, board)
	})
	return board, err
}
