package service

import (
	"context"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"strings"
	"testing"
)

// directoryOrder 返回父级下未归档项的名称，按目录顺序。
func directoryOrder(t *testing.T, userID int64, parentID *int64) []string {
	t.Helper()
	siblings, err := activeSiblings(context.Background(), userID, parentID, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(siblings))
	for _, s := range siblings {
		if s.folder != nil {
			names = append(names, s.folder.Name)
		} else {
			names = append(names, s.board.Name)
		}
	}
	return names
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDirectoryOrdering(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")

		mustBoard := func(name string, index *int) {
			t.Helper()
			if _, err := Board.Create(ctx, alice.ID, nil, name, index); err != nil {
				t.Fatal(err)
			}
		}
		mustBoard("A", nil)
		if _, err := Folder.Create(ctx, alice.ID, nil, "F", nil); err != nil {
			t.Fatal(err)
		}
		mustBoard("B", ptr(0)) // 后建的看板可以排到最前
		if got := directoryOrder(t, alice.ID, nil); !equalNames(got, []string{"B", "A", "F"}) {
			t.Fatalf("顺序 = %v", got)
		}

		// 反复插到第 2 位，间隙用完后自动重排，顺序仍然正确。
		want := []string{"B", "A", "F"}
		for i := range 20 {
			name := string(rune('a' + i))
			mustBoard(name, ptr(1))
			want = append([]string{want[0], name}, want[1:]...)
		}
		if got := directoryOrder(t, alice.ID, nil); !equalNames(got, want) {
			t.Fatalf("多次插入后顺序 = %v，期望 %v", got, want)
		}
	})
}

func TestFolderDepthAndMove(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")

		var parent *int64
		var chain []int64
		for range maxFolderDepth {
			folder, err := Folder.Create(ctx, alice.ID, parent, "层", nil)
			if err != nil {
				t.Fatalf("第 %d 层创建失败: %v", len(chain)+1, err)
			}
			chain = append(chain, folder.ID)
			parent = &folder.ID
		}
		if _, err := Folder.Create(ctx, alice.ID, parent, "第 17 层", nil); errorKind(err) != KindBadRequest {
			t.Fatalf("第 17 层 err = %v", err)
		}
		// 看板放在最深的文件夹里不增加深度。
		if _, err := Board.Create(ctx, alice.ID, parent, "看板", nil); err != nil {
			t.Fatal(err)
		}

		other, err := Folder.Create(ctx, alice.ID, nil, "另一个", nil)
		if err != nil {
			t.Fatal(err)
		}
		// 把只有一层的文件夹移到第 16 层下会超过 16 层；移到第 15 层下刚好 16 层。
		if _, err = Folder.Move(ctx, alice.ID, other.ID, &chain[15], nil); errorKind(err) != KindBadRequest {
			t.Fatalf("移到第 16 层下 err = %v", err)
		}
		if _, err = Folder.Move(ctx, alice.ID, other.ID, &chain[14], nil); err != nil {
			t.Fatalf("移到第 15 层下 err = %v", err)
		}
		// 不能移到自己的下级中。
		if _, err = Folder.Move(ctx, alice.ID, chain[0], &chain[3], nil); errorKind(err) != KindBadRequest {
			t.Fatalf("移到自己的下级中 err = %v", err)
		}
	})
}

func TestArchiveAndRestore(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")

		work, _ := Folder.Create(ctx, alice.ID, nil, "工作", nil)
		project, _ := Folder.Create(ctx, alice.ID, &work.ID, "项目", nil)
		early, _ := Board.Create(ctx, alice.ID, &project.ID, "早先归档", nil)
		main, _ := Board.Create(ctx, alice.ID, &project.ID, "主看板", nil)
		home, _ := Board.Create(ctx, alice.ID, nil, "生活", nil)
		if err := Board.Archive(ctx, alice.ID, early.ID); err != nil {
			t.Fatal(err)
		}
		earlyArchived, _ := repository.Board.FindByID(ctx, alice.ID, early.ID)
		if _, err := Board.SetMain(ctx, alice.ID, &main.ID); err != nil {
			t.Fatal(err)
		}

		// 归档文件夹：下级文件夹和其中的看板一起归档，主看板指向清空，早先归档的看板保留原归档时间。
		if err := Folder.Archive(ctx, alice.ID, work.ID); err != nil {
			t.Fatal(err)
		}
		for _, id := range []int64{work.ID, project.ID} {
			folder, _ := repository.Folder.FindByID(ctx, alice.ID, id)
			if !folder.Archived() {
				t.Fatalf("文件夹 %d 未归档", id)
			}
		}
		mainBoard, _ := repository.Board.FindByID(ctx, alice.ID, main.ID)
		if !mainBoard.Archived() || mainBoard.FolderID == nil || *mainBoard.FolderID != project.ID {
			t.Fatalf("主看板 = %+v，应已归档并保留原文件夹", mainBoard)
		}
		earlyAfter, _ := repository.Board.FindByID(ctx, alice.ID, early.ID)
		if !earlyAfter.ArchivedAt.Equal(*earlyArchived.ArchivedAt) {
			t.Fatal("早先归档的看板的归档时间被改写")
		}
		user, _ := User.FindByID(ctx, alice.ID)
		if user.MainBoardID != nil {
			t.Fatal("主看板归档后指向未清空")
		}
		if got := directoryOrder(t, alice.ID, nil); !equalNames(got, []string{"生活"}) {
			t.Fatalf("目录 = %v", got)
		}

		// 已归档的对象不能再操作，也不能设为主看板或作为目标父级。
		if _, err := Board.SetMain(ctx, alice.ID, &main.ID); errorKind(err) != KindBadRequest {
			t.Fatalf("把归档看板设为主看板 err = %v", err)
		}
		if _, err := Board.Restore(ctx, alice.ID, main.ID, &project.ID, nil); errorKind(err) != KindBadRequest {
			t.Fatalf("恢复到已归档的文件夹 err = %v", err)
		}
		if _, err := Board.Restore(ctx, alice.ID, home.ID, nil, nil); errorKind(err) != KindBadRequest {
			t.Fatalf("恢复未归档的看板 err = %v", err)
		}

		// 恢复到根的最前面：只恢复这一个看板，原文件夹仍在归档中。
		restored, err := Board.Restore(ctx, alice.ID, main.ID, nil, ptr(0))
		if err != nil {
			t.Fatal(err)
		}
		if restored.Archived() || restored.FolderID != nil {
			t.Fatalf("恢复后的看板 = %+v", restored)
		}
		if got := directoryOrder(t, alice.ID, nil); !equalNames(got, []string{"主看板", "生活"}) {
			t.Fatalf("恢复后目录 = %v", got)
		}
		if folder, _ := repository.Folder.FindByID(ctx, alice.ID, project.ID); !folder.Archived() {
			t.Fatal("恢复看板时一并恢复了文件夹")
		}
	})
}

func TestDirectoryIsolatedByUser(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		bob := mustUser(t, "bob")
		folder, _ := Folder.Create(ctx, alice.ID, nil, "alice 的文件夹", nil)
		board, _ := Board.Create(ctx, alice.ID, nil, "alice 的看板", nil)

		if _, err := Folder.Rename(ctx, bob.ID, folder.ID, "改名"); errorKind(err) != KindNotFound {
			t.Fatalf("改其他用户的文件夹 err = %v", err)
		}
		if _, err := Board.Create(ctx, bob.ID, &folder.ID, "放进别人的文件夹", nil); errorKind(err) != KindNotFound {
			t.Fatalf("在其他用户的文件夹中建看板 err = %v", err)
		}
		if err := Board.Archive(ctx, bob.ID, board.ID); errorKind(err) != KindNotFound {
			t.Fatalf("归档其他用户的看板 err = %v", err)
		}
		if _, err := Board.SetMain(ctx, bob.ID, &board.ID); errorKind(err) != KindNotFound {
			t.Fatalf("把其他用户的看板设为主看板 err = %v", err)
		}
		catchUp, err := Sync.CatchUp(ctx, bob.ID, 0)
		if err != nil || len(catchUp.Snapshot.Folders) != 0 || len(catchUp.Snapshot.Boards) != 0 {
			t.Fatalf("bob 的快照 = %+v, %v", catchUp, err)
		}
	})
}

func TestValidateName(t *testing.T) {
	if name, err := validateName("  看板  "); err != nil || name != "看板" {
		t.Fatalf("validateName = %q, %v", name, err)
	}
	for _, name := range []string{"", "   ", strings.Repeat("长", 101)} {
		if _, err := validateName(name); errorKind(err) != KindBadRequest {
			t.Fatalf("validateName(%q) err = %v", name, err)
		}
	}
}
