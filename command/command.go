package command

import (
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	boardproto "github.com/liam-lai/ptt-alertor/models/ptt/board"
	"github.com/liam-lai/ptt-alertor/models/subscription"
	user "github.com/liam-lai/ptt-alertor/models/user/redis"
)

var Commands = map[string]map[string]string{
	"一般": {
		"指令": "可使用的指令清單",
		"清單": "設定的看板、關鍵字、作者",
	},
	"關鍵字相關": {
		"新增 看板 關鍵字": "新增看板關鍵字。",
		"刪除 看板 關鍵字": "刪除看板關鍵字。",
	},
	"作者相關": {
		"新增作者 看板 作者": "新增看板作者。",
		"刪除作者 看板 作者": "刪除看板作者。",
	},
	"範例": {
		"新增": "新增 nba,lol 樂透,情報",
		"刪除": "刪除 nba 樂透",
		"作者": "新增作者 gossiping ffaarr,obov",
	},
}

func HandleCommand(text string, userID string) string {
	command := strings.Fields(strings.TrimSpace(text))[0]
	log.WithFields(log.Fields{
		"account": userID,
		"command": command,
	}).Info("Command Request")
	switch command {
	case "清單":
		var rspText string
		subs := new(user.User).Find(userID).Subscribes
		if len(subs) == 0 {
			rspText = "尚未建立清單。請打「指令」查看新增方法。"
		} else {
			rspText = new(user.User).Find(userID).Subscribes.String()
		}
		return rspText
	case "指令":
		return stringCommands()
	case "新增", "刪除", "新增作者", "刪除作者":
		re := regexp.MustCompile("^(新增|新增作者|刪除|刪除作者)\\s+([^,，][\\w\\d-_,，]+[^,，])\\s+(.+)")
		matched := re.MatchString(text)
		if !matched {
			if strings.Contains(command, "作者") {
				return "指令格式錯誤。\n1.板名欄位開頭與結尾不可有逗號\n2.板名欄位間不允許空白字元。\n正確範例：" + command + " gossiping,lol ffaarr,obov"
			}
			return "指令格式錯誤。\n1.板名欄位開頭與結尾不可有逗號\n2.板名欄位間不允許空白字元。\n正確範例：" + command + " gossiping,lol 問卦,爆卦"
		}
		args := re.FindStringSubmatch(text)
		boardNames := splitParamString(args[2])
		keywords := splitParamString(args[3])
		var err error
		if command == "新增" || command == "新增作者" {
			if command == "新增" {
				err = update(userID, boardNames, keywords, addKeywords)
			} else if command == "新增作者" {
				err = update(userID, boardNames, keywords, addAuthors)
			}
			if bErr, ok := err.(boardproto.BoardNotExistError); ok {
				return "版名錯誤，請確認拼字。可能版名：\n" + bErr.Suggestion
			}
			if err != nil {
				return "新增失敗，請等待修復。"
			}
			return "新增成功"
		}
		if command == "刪除" || command == "刪除作者" {
			if command == "刪除" {
				err = update(userID, boardNames, keywords, removeKeywords)
			} else if command == "刪除作者" {
				err = update(userID, boardNames, keywords, removeAuthors)
			}
			if bErr, ok := err.(boardproto.BoardNotExistError); ok {
				return "版名錯誤，請確認拼字。可能版名：\n" + bErr.Suggestion
			}
			if err != nil {
				return "刪除失敗，請等待修復。"
			}
		}
		return "刪除成功"
	}
	return "無此指令，請打「指令」查看指令清單"
}

func stringCommands() string {
	str := ""
	for cat, cmds := range Commands {
		str += "[" + cat + "]\n"
		for cmd, doc := range cmds {
			str += cmd + "：" + doc + "\n"
		}
		str += "\n"
	}
	return str
}

func splitParamString(paramString string) (params []string) {

	paramString = strings.Trim(paramString, ",，")

	if !strings.ContainsAny(paramString, ",，") {
		return []string{paramString}
	}

	if strings.Contains(paramString, ",") {
		params = strings.Split(paramString, ",")
	} else {
		params = []string{paramString}
	}

	for i := 0; i < len(params); i++ {
		if strings.Contains(params[i], "，") {
			params = append(params[:i], append(strings.Split(params[i], "，"), params[i+1:]...)...)
			i--
		}
	}

	for i, param := range params {
		params[i] = strings.TrimSpace(param)
	}

	return params
}

func update(account string, boardNames []string, inputs []string, action updateAction) error {
	for _, boardName := range boardNames {
		u := new(user.User).Find(account)
		sub := subscription.Subscription{
			Board: boardName,
		}
		err := action(&u, sub, inputs)
		if err != nil {
			return err
		}
		err = u.Update()
		if err != nil {
			log.WithError(err).Error("Subscription Update Error")
			return err
		}
	}
	return nil
}

func HandleLineFollow(id string) error {
	u := new(user.User).Find(id)
	u.Profile.Line = id
	return handleFollow(u)
}

func HandleMessengerFollow(id string) error {
	u := new(user.User).Find(id)
	u.Profile.Messenger = id
	return handleFollow(u)
}

func handleFollow(u user.User) error {
	if u.Profile.Account != "" {
		u.Enable = true
		u.Update()
	} else {
		if u.Profile.Messenger != "" {
			u.Profile.Account = u.Profile.Messenger
		} else {
			u.Profile.Account = u.Profile.Line
		}
		u.Enable = true
		err := u.Save()
		if err != nil {
			return err
		}
	}
	return nil
}