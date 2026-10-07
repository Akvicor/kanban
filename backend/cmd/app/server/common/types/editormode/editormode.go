package editormode

// Type 是卡片描述编辑器的模式，与前端 src/markdown/MarkdownEditor.tsx 的取值一致。
// 用户在编辑器中切换模式后保存，下次编辑描述时直接进入该模式。
type Type string

const (
	Wysiwyg Type = "wysiwyg" // 所见即所得，新用户默认
	Markup  Type = "markup"  // Markdown 源码
)

// Default 是新用户的编辑器模式。
const Default = Wysiwyg

// Valid 判断编辑器模式取值是否合法。
func (t Type) Valid() bool {
	return t == Wysiwyg || t == Markup
}
