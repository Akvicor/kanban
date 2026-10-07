package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"

	"gorm.io/gorm"
)

// CardLink 是卡片关联的服务。关联只是从卡片快速跳转到某个看板或面板的入口，目标可以是任何看板或面板，
// 包括卡片自己所在的，也可以已经归档。
var CardLink = new(cardLinkService)

type cardLinkService struct{}

// Add 给卡片加一条关联，boardID 和 panelID 只能提供一个。同一卡片对同一目标只保留一条，按添加顺序排在最后。
func (s *cardLinkService) Add(ctx context.Context, userID, cardID int64, boardID, panelID *int64) (*model.CardLink, error) {
	if (boardID == nil) == (panelID == nil) {
		return nil, badRequest(resp.CardLinkTargetInvalid, "关联目标必须是一个看板或一个面板")
	}
	var link *model.CardLink
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, _, err := usableCard(ctx, userID, cardID); err != nil {
			return err
		}
		if boardID != nil {
			if _, err := findBoard(ctx, userID, *boardID); err != nil {
				return err
			}
		} else if _, err := findPanel(ctx, userID, *panelID); err != nil {
			return err
		}
		links, err := repository.CardLink.ListByCards(ctx, userID, []int64{cardID})
		if err != nil {
			return err
		}
		positions := make([]int64, 0, len(links))
		for _, existing := range links {
			if sameTarget(existing.BoardID, boardID) && sameTarget(existing.PanelID, panelID) {
				return conflict(resp.CardLinkExists, "已经关联了这个目标")
			}
			positions = append(positions, existing.Position)
		}
		position, _ := slotPosition(positions, nil)
		link = &model.CardLink{UserID: userID, CardID: cardID, BoardID: boardID, PanelID: panelID, Position: position}
		if err = repository.CardLink.Create(ctx, link); err != nil {
			return err
		}
		return recordCardLink(ctx, link)
	})
	return link, err
}

func sameTarget(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Delete 从卡片上删除一条关联，不影响目标。
func (s *cardLinkService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		link, err := repository.CardLink.FindByID(ctx, userID, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound(resp.CardLinkNotFound, "关联不存在")
		}
		if err != nil {
			return err
		}
		if _, _, err = usableCard(ctx, userID, link.CardID); err != nil {
			return err
		}
		if err = repository.CardLink.Delete(ctx, userID, id); err != nil {
			return err
		}
		return recordChange(ctx, userID, hub.OpDelete, EntityCardLink, id, nil)
	})
}
