package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

// priorityNameMaxLength 是优先级挡位名称的最大字符数。
const priorityNameMaxLength = 20

// PriorityLevel 是面板中优先级挡位的服务。只能修改正常使用中的面板的挡位。
var PriorityLevel = new(priorityLevelService)

type priorityLevelService struct{}

func validatePriorityName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", badRequest(resp.PriorityLevelNameEmpty, "挡位名称不能为空")
	}
	if utf8.RuneCountInString(name) > priorityNameMaxLength {
		return "", badRequest(resp.PriorityLevelNameTooLong, "挡位名称不能超过 20 个字符")
	}
	return name, nil
}

func recordPriorityLevel(ctx context.Context, level *model.PriorityLevel) error {
	return recordChange(ctx, level.UserID, hub.OpUpsert, EntityPriorityLevel, level.ID, dro.NewPriorityLevel(level))
}

// usablePriorityLevel 查找挡位，并确认它所在的面板正常使用中。
func usablePriorityLevel(ctx context.Context, userID, id int64) (*model.PriorityLevel, error) {
	level, err := repository.PriorityLevel.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.PriorityLevelNotFound, "优先级挡位不存在")
	}
	if err != nil {
		return nil, err
	}
	if _, _, err = usablePanel(ctx, userID, level.PanelID); err != nil {
		return nil, err
	}
	return level, nil
}

// Create 在面板中新建挡位，排在最后（优先级最低）。
func (s *priorityLevelService) Create(ctx context.Context, userID, panelID int64, name, color string) (*model.PriorityLevel, error) {
	name, err := validatePriorityName(name)
	if err != nil {
		return nil, err
	}
	if color, err = validateColor(color); err != nil {
		return nil, err
	}
	level := &model.PriorityLevel{UserID: userID, PanelID: panelID, Name: name, Color: color}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, _, err := usablePanel(ctx, userID, panelID); err != nil {
			return err
		}
		levels, err := repository.PriorityLevel.ListByPanel(ctx, userID, panelID)
		if err != nil {
			return err
		}
		if len(levels) > 0 {
			level.Position = levels[len(levels)-1].Position + positionGap
		} else {
			level.Position = positionGap
		}
		if err = repository.PriorityLevel.Create(ctx, level); err != nil {
			return err
		}
		return recordPriorityLevel(ctx, level)
	})
	if err != nil {
		return nil, err
	}
	return level, nil
}

// Update 修改挡位的名称和颜色。
func (s *priorityLevelService) Update(ctx context.Context, userID, id int64, name, color string) (*model.PriorityLevel, error) {
	name, err := validatePriorityName(name)
	if err != nil {
		return nil, err
	}
	if color, err = validateColor(color); err != nil {
		return nil, err
	}
	var level *model.PriorityLevel
	err = write(ctx, func(ctx context.Context) error {
		if level, err = usablePriorityLevel(ctx, userID, id); err != nil {
			return err
		}
		level.Name, level.Color = name, color
		if err = repository.PriorityLevel.Update(ctx, userID, id, map[string]any{"name": name, "color": color}); err != nil {
			return err
		}
		return recordPriorityLevel(ctx, level)
	})
	return level, err
}

// Reorder 把挡位移到面板中第 index 位，index 为 nil 时放到最后。按优先级排序时使用这个顺序。
func (s *priorityLevelService) Reorder(ctx context.Context, userID, id int64, index *int) (*model.PriorityLevel, error) {
	var level *model.PriorityLevel
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if level, err = usablePriorityLevel(ctx, userID, id); err != nil {
			return err
		}
		levels, err := repository.PriorityLevel.ListByPanel(ctx, userID, level.PanelID)
		if err != nil {
			return err
		}
		save := func(ctx context.Context, l *model.PriorityLevel, value int64) error {
			l.Position = value
			if err := repository.PriorityLevel.Update(ctx, userID, l.ID, map[string]any{"position": value}); err != nil {
				return err
			}
			return recordPriorityLevel(ctx, l)
		}
		position, err := placeInList(ctx, levels, id, index,
			func(l *model.PriorityLevel) int64 { return l.ID },
			func(l *model.PriorityLevel) int64 { return l.Position },
			save)
		if err != nil {
			return err
		}
		return save(ctx, level, position)
	})
	return level, err
}

// Delete 彻底删除挡位，不能恢复。使用它的卡片（包括归档中的）改为没有优先级。
func (s *priorityLevelService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		level, err := usablePriorityLevel(ctx, userID, id)
		if err != nil {
			return err
		}
		cards, err := repository.Card.ListByPriorityLevel(ctx, userID, id)
		if err != nil {
			return err
		}
		for _, card := range cards {
			card.PriorityLevelID = nil
			if err = repository.Card.Update(ctx, userID, card.ID, map[string]any{"priority_level_id": nil}); err != nil {
				return err
			}
			if err = recordCard(ctx, card); err != nil {
				return err
			}
		}
		if err = repository.PriorityLevel.Delete(ctx, userID, id); err != nil {
			return err
		}
		return recordChange(ctx, userID, hub.OpDelete, EntityPriorityLevel, level.ID, nil)
	})
}
