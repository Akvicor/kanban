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

// labelNameMaxLength 是标签名称的最大字符数。标签名称可以为空，此时只显示颜色。
const labelNameMaxLength = 50

// Label 是面板中标签的服务。只能修改正常使用中的面板的标签。
var Label = new(labelService)

type labelService struct{}

func validateLabelName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > labelNameMaxLength {
		return "", badRequest(resp.LabelNameTooLong, "标签名称不能超过 50 个字符")
	}
	return name, nil
}

func recordLabel(ctx context.Context, label *model.Label) error {
	return recordChange(ctx, label.UserID, hub.OpUpsert, EntityLabel, label.ID, dro.NewLabel(label))
}

// usableLabel 查找标签，并确认它所在的面板正常使用中。
func usableLabel(ctx context.Context, userID, id int64) (*model.Label, error) {
	label, err := repository.Label.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.LabelNotFound, "标签不存在")
	}
	if err != nil {
		return nil, err
	}
	if _, _, err = usablePanel(ctx, userID, label.PanelID); err != nil {
		return nil, err
	}
	return label, nil
}

// Create 在面板中新建标签，排在最后。
func (s *labelService) Create(ctx context.Context, userID, panelID int64, name, color string) (*model.Label, error) {
	name, err := validateLabelName(name)
	if err != nil {
		return nil, err
	}
	if color, err = validateColor(color); err != nil {
		return nil, err
	}
	label := &model.Label{UserID: userID, PanelID: panelID, Name: name, Color: color}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, _, err := usablePanel(ctx, userID, panelID); err != nil {
			return err
		}
		labels, err := repository.Label.ListByPanel(ctx, userID, panelID)
		if err != nil {
			return err
		}
		if len(labels) > 0 {
			label.Position = labels[len(labels)-1].Position + positionGap
		} else {
			label.Position = positionGap
		}
		if err = repository.Label.Create(ctx, label); err != nil {
			return err
		}
		return recordLabel(ctx, label)
	})
	if err != nil {
		return nil, err
	}
	return label, nil
}

// Update 修改标签的名称和颜色。卡片和列表操作配置通过标签 ID 引用，跟着生效。
func (s *labelService) Update(ctx context.Context, userID, id int64, name, color string) (*model.Label, error) {
	name, err := validateLabelName(name)
	if err != nil {
		return nil, err
	}
	if color, err = validateColor(color); err != nil {
		return nil, err
	}
	var label *model.Label
	err = write(ctx, func(ctx context.Context) error {
		if label, err = usableLabel(ctx, userID, id); err != nil {
			return err
		}
		label.Name, label.Color = name, color
		if err = repository.Label.Update(ctx, userID, id, map[string]any{"name": name, "color": color}); err != nil {
			return err
		}
		return recordLabel(ctx, label)
	})
	return label, err
}

// Reorder 把标签移到面板中第 index 位，index 为 nil 时放到最后。
func (s *labelService) Reorder(ctx context.Context, userID, id int64, index *int) (*model.Label, error) {
	var label *model.Label
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if label, err = usableLabel(ctx, userID, id); err != nil {
			return err
		}
		labels, err := repository.Label.ListByPanel(ctx, userID, label.PanelID)
		if err != nil {
			return err
		}
		save := func(ctx context.Context, l *model.Label, value int64) error {
			l.Position = value
			if err := repository.Label.Update(ctx, userID, l.ID, map[string]any{"position": value}); err != nil {
				return err
			}
			return recordLabel(ctx, l)
		}
		position, err := placeInList(ctx, labels, id, index,
			func(l *model.Label) int64 { return l.ID },
			func(l *model.Label) int64 { return l.Position },
			save)
		if err != nil {
			return err
		}
		return save(ctx, label, position)
	})
	return label, err
}

// Delete 彻底删除标签，不能恢复。同时从卡片上移除该标签、删除引用它的列表操作配置，
// 包括归档中的列表和卡片，不回头重算其他操作。
func (s *labelService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		label, err := usableLabel(ctx, userID, id)
		if err != nil {
			return err
		}
		listIDs, err := repository.List.ListIDsUsingLabel(ctx, userID, id)
		if err != nil {
			return err
		}
		if err = repository.List.DeleteLabelRules(ctx, userID, id); err != nil {
			return err
		}
		for _, listID := range listIDs {
			list, err := findList(ctx, userID, listID)
			if err != nil {
				return err
			}
			if err = recordList(ctx, list); err != nil {
				return err
			}
		}
		cardIDs, err := repository.Card.CardIDsWithLabel(ctx, userID, id)
		if err != nil {
			return err
		}
		if err = repository.Card.DeleteLabel(ctx, userID, id); err != nil {
			return err
		}
		for _, cardID := range cardIDs {
			card, err := findCard(ctx, userID, cardID)
			if err != nil {
				return err
			}
			if err = recordCard(ctx, card); err != nil {
				return err
			}
		}
		if err = repository.Label.Delete(ctx, userID, id); err != nil {
			return err
		}
		return recordChange(ctx, userID, hub.OpDelete, EntityLabel, label.ID, nil)
	})
}
