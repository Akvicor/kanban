package service

import "testing"

func TestSlotPosition(t *testing.T) {
	g := positionGap
	cases := []struct {
		name      string
		positions []int64
		index     *int
		want      int64
		renumber  map[int]int64
	}{
		{"空列表", nil, nil, g, nil},
		{"末尾", []int64{g, 2 * g}, nil, 3 * g, nil},
		{"超出范围放末尾", []int64{g}, ptr(5), 2 * g, nil},
		{"最前", []int64{g, 2 * g}, ptr(0), 0, nil},
		{"中间", []int64{g, 2 * g}, ptr(1), g + g/2, nil},
		{"没有空隙时重排", []int64{10, 11, 12}, ptr(1), 2 * g, map[int]int64{0: g, 1: 3 * g, 2: 4 * g}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, renumber := slotPosition(c.positions, c.index)
			if got != c.want || len(renumber) != len(c.renumber) {
				t.Fatalf("slotPosition = %d, %v；期望 %d, %v", got, renumber, c.want, c.renumber)
			}
			for i, v := range c.renumber {
				if renumber[i] != v {
					t.Fatalf("第 %d 项 = %d，期望 %d", i, renumber[i], v)
				}
			}
		})
	}
}
