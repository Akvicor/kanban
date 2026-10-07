package service

import "context"

// placeInList 计算把 movingID 放到 items（已按顺序排列）第 index 位时的排序值，index 为 nil 时放在末尾。
// items 中与 movingID 相同的项不参与计算；需要重新排列时，通过 save 修改其他项的排序值。
func placeInList[T any](
	ctx context.Context,
	items []*T,
	movingID int64,
	index *int,
	idOf func(*T) int64,
	positionOf func(*T) int64,
	save func(context.Context, *T, int64) error,
) (int64, error) {
	others := make([]*T, 0, len(items))
	positions := make([]int64, 0, len(items))
	for _, item := range items {
		if idOf(item) != movingID {
			others = append(others, item)
			positions = append(positions, positionOf(item))
		}
	}
	position, renumber := slotPosition(positions, index)
	for i, item := range others {
		if value, ok := renumber[i]; ok {
			if err := save(ctx, item, value); err != nil {
				return 0, err
			}
		}
	}
	return position, nil
}

// positionGap 是新排序值之间的间隙。插入时取前后两项的中间值，间隙用完时把同级重新等距排列。
// 目录、面板、标签和优先级挡位都按这个规则排序。
const positionGap int64 = 1 << 16

// slotPosition 计算把一项插到第 index 位（从 0 开始）时的排序值。
// positions 是同级其他项按顺序排列的排序值，不含被插入的项；index 为 nil 或超出范围时放在末尾。
// 前后两项之间没有空隙时，同级重新等距排列：renumber 给出需要修改的项（下标 → 新排序值）。
func slotPosition(positions []int64, index *int) (position int64, renumber map[int]int64) {
	if index == nil || *index >= len(positions) {
		if len(positions) == 0 {
			return positionGap, nil
		}
		return positions[len(positions)-1] + positionGap, nil
	}
	at := max(*index, 0)
	upper := positions[at]
	lower := upper - 2*positionGap
	if at > 0 {
		lower = positions[at-1]
	}
	if upper-lower >= 2 {
		return lower + (upper-lower)/2, nil
	}

	renumber = map[int]int64{}
	order := int64(0)
	next := func() int64 {
		order++
		return order * positionGap
	}
	for i, current := range positions {
		if i == at {
			position = next()
		}
		if value := next(); value != current {
			renumber[i] = value
		}
	}
	return position, renumber
}
