package config

import (
	"bytes"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// marshalWithComments 把 v 编码为 YAML，并把结构体字段的 comment 标签写成对应键上方的注释。
// yaml.v3 本身不读取 comment 标签，这里先编码为节点树，再按字段类型逐层附加注释。
func marshalWithComments(v any) ([]byte, error) {
	var node yaml.Node
	if err := node.Encode(v); err != nil {
		return nil, err
	}
	attachComments(&node, reflect.TypeOf(v))

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// attachComments 递归处理映射节点：键名按 yaml 标签匹配到字段，再把字段的 comment 标签写到键上。
func attachComments(node *yaml.Node, t reflect.Type) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			attachComments(child, t)
		}
		return
	}
	if node.Kind != yaml.MappingNode || t.Kind() != reflect.Struct {
		return
	}

	fields := make(map[string]reflect.StructField, t.NumField())
	for i := range t.NumField() {
		field := t.Field(i)
		if name, _, _ := strings.Cut(field.Tag.Get("yaml"), ","); name != "" {
			fields[name] = field
		}
	}
	// 映射节点的 Content 按「键、值、键、值」交替排列。
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		field, ok := fields[key.Value]
		if !ok {
			continue
		}
		if comment := field.Tag.Get("comment"); comment != "" {
			key.HeadComment = comment
		}
		attachComments(value, field.Type)
	}
}
