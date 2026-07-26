//go:build windows

package main

import (
	"fmt"
	"strings"
)

func validateExecutableOutputItems(items []Item) error {
	for _, item := range items {
		switch {
		case strings.EqualFold(item.Kind, "Key"):
			if _, ok := parseVK(item.Code); !ok {
				return fmt.Errorf("不明なキーです: %s", item.Code)
			}
		case strings.EqualFold(item.Kind, "Mouse"):
			if _, ok := joyConMouseTapInputs(item.Code); !ok {
				return fmt.Errorf("不明なマウス操作です: %s", item.Code)
			}
		case strings.EqualFold(item.Kind, "JoyCon"):
			return fmt.Errorf("Joy-Conは入力側にだけ指定できます")
		default:
			return fmt.Errorf("未対応の実行内容です: %s:%s", item.Kind, item.Code)
		}
	}
	return nil
}
