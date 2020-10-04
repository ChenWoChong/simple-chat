package utils

import (
	"errors"
	"strings"
)

func ParseContent(str string) (userName string, content string, err error) {

	index := strings.Index(str, `@`)
	if index != 0 {
		return "", str, nil
	}

	strList := strings.Split(str, " ")
	if len(strList) <= 1 {
		return "", str, errors.New("[Bad format] 私聊: @用户名[空格]聊天内容")
	} else if strList[0] == `@` {
		return "", str, errors.New("[Bad format] 私聊: @用户名[空格]聊天内容")
	}

	userName = strings.TrimPrefix(strList[0], "@")
	content = strings.Join(strList[1:], " ")

	return userName, content, nil
}
