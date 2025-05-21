package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"start-feishubot/initialization"
	"start-feishubot/services"
	"strings"
)

type CardKind string
type CardChatType string

var (
	ClearCardKind     = CardKind("clear")           // 清空上下文
	PicModeChangeKind = CardKind("pic_mode_change") // 切换图片创作模式
	PicResolutionKind = CardKind("pic_resolution")  // 图片分辨率调整
	PicTextMoreKind   = CardKind("pic_text_more")   // 重新根据文本生成图片
	PicVarMoreKind    = CardKind("pic_var_more")    // 变量图片
)

var (
	GroupChatType = CardChatType("group")
	UserChatType  = CardChatType("personal")
)

type CardMsg struct {
	Kind      CardKind
	ChatType  CardChatType
	Value     interface{}
	SessionId string
	MsgId     string
}

type MenuOption struct {
	value string
	label string
}

func replyCard(ctx context.Context,
	msgId *string,
	cardContent string,
) error {
	client := initialization.GetLarkClient()
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Uuid(uuid.New().String()).
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return err
	}
	return nil
}

func newSendCard(
	header *larkcard.MessageCardHeader,
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	for _, element := range elements {
		aElementPool = append(aElementPool, element)
	}
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Header(header).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

func newSimpleSendCard(
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	for _, element := range elements {
		aElementPool = append(aElementPool, element)
	}
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

// withSplitLine 用于生成分割线
func withSplitLine() larkcard.MessageCardElement {
	splitLine := larkcard.NewMessageCardHr().
		Build()
	return splitLine
}

// withHeader 用于生成消息头
func withHeader(title string, color string) *larkcard.
MessageCardHeader {
	if title == "" {
		title = "🤖️机器人提醒"
	}
	header := larkcard.NewMessageCardHeader().
		Template(color).
		Title(larkcard.NewMessageCardPlainText().
			Content(title).
			Build()).
		Build()
	return header
}

// withNote 用于生成纯文本脚注
func withNote(note string) larkcard.MessageCardElement {
	noteElement := larkcard.NewMessageCardNote().
		Elements([]larkcard.MessageCardNoteElement{larkcard.NewMessageCardPlainText().
			Content(note).
			Build()}).
		Build()
	return noteElement
}

// withMainMd 用于生成markdown消息体
func withMainMd(msg string) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = processNewLine(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardLarkMd().
				Content(msg).
				Build()).
			IsShort(true).
			Build()}).
		Build()
	return mainElement
}

// withMainText 用于生成纯文本消息体
func withMainText(msg string) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = cleanTextBlock(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardPlainText().
				Content(msg).
				Build()).
			IsShort(false).
			Build()}).
		Build()
	return mainElement
}

func withImageDiv(imageKey string) larkcard.MessageCardElement {
	imageElement := larkcard.NewMessageCardImage().
		ImgKey(imageKey).
		Alt(larkcard.NewMessageCardPlainText().Content("").
			Build()).
		Preview(true).
		Mode(larkcard.MessageCardImageModelCropCenter).
		CompactWidth(true).
		Build()
	return imageElement
}

// withMdAndExtraBtn 用于生成带有额外按钮的消息体
func withMdAndExtraBtn(msg string, btn *larkcard.
MessageCardEmbedButton) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = processNewLine(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields(
			[]*larkcard.MessageCardField{
				larkcard.NewMessageCardField().
					Text(larkcard.NewMessageCardLarkMd().
						Content(msg).
						Build()).
					IsShort(true).
					Build()}).
		Extra(btn).
		Build()
	return mainElement
}

func newBtn(content string, value map[string]interface{},
	typename larkcard.MessageCardButtonType) *larkcard.
MessageCardEmbedButton {
	btn := larkcard.NewMessageCardEmbedButton().
		Type(typename).
		Value(value).
		Text(larkcard.NewMessageCardPlainText().
			Content(content).
			Build())
	return btn
}

func newMenu(
	placeHolder string,
	value map[string]interface{},
	options ...MenuOption,
) *larkcard.
MessageCardEmbedSelectMenuStatic {
	var aOptionPool []*larkcard.MessageCardEmbedSelectOption
	for _, option := range options {
		aOption := larkcard.NewMessageCardEmbedSelectOption().
			Value(option.value).
			Text(larkcard.NewMessageCardPlainText().
				Content(option.label).
				Build())
		aOptionPool = append(aOptionPool, aOption)

	}
	btn := larkcard.NewMessageCardEmbedSelectMenuStatic().
		MessageCardEmbedSelectMenuStatic(larkcard.NewMessageCardEmbedSelectMenuBase().
			Options(aOptionPool).
			Placeholder(larkcard.NewMessageCardPlainText().
				Content(placeHolder).
				Build()).
			Value(value).
			Build()).
		Build()
	return btn
}

// 清除卡片按钮
func withClearDoubleCheckBtn(sessionID *string) larkcard.MessageCardElement {
	confirmBtn := newBtn("确认清除", map[string]interface{}{
		"value":     "1",
		"kind":      ClearCardKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"value":     "0",
		"kind":      ClearCardKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withPicModeDoubleCheckBtn(sessionID *string) larkcard.
MessageCardElement {
	confirmBtn := newBtn("切换模式", map[string]interface{}{
		"value":     "1",
		"kind":      PicModeChangeKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn("我再想想", map[string]interface{}{
		"value":     "0",
		"kind":      PicModeChangeKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withOneBtn(btn *larkcard.MessageCardEmbedButton) larkcard.
MessageCardElement {
	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{btn}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

//新建对话按钮

func withPicResolutionBtn(sessionID *string) larkcard.
MessageCardElement {
	cancelMenu := newMenu("默认分辨率",
		map[string]interface{}{
			"value":     "0",
			"kind":      PicResolutionKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		MenuOption{
			label: "256x256",
			value: string(services.Resolution256),
		},
		MenuOption{
			label: "512x512",
			value: string(services.Resolution512),
		},
		MenuOption{
			label: "1024x1024",
			value: string(services.Resolution1024),
		},
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()

	return actions

}

//func replyMsg(ctx context.Context, msg string, msgId *string) error {
//	fmt.Println("sendMsg", msg, msgId)
//	msg, i := processMessage(msg)
//	if i != nil {
//		return i
//	}
//	client := initialization.GetLarkClient()
//	content := larkim.NewTextMsgBuilder().
//		Text(msg).
//		Build()
//
//	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
//		MessageId(*msgId).
//		Body(larkim.NewReplyMessageReqBodyBuilder().
//			MsgType(larkim.MsgTypeText).
//			Uuid(uuid.New().String()).
//			Content(content).
//			Build()).
//		Build())
//
//	// 处理错误
//	if err != nil {
//		fmt.Println(err)
//		return err
//	}
//
//	// 服务端错误处理
//	if !resp.Success() {
//		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
//		return err
//	}
//	return nil
//}

func replyMsg(ctx context.Context, msg string, msgId *string) error {
	fmt.Println("sendMsg", msg, msgId)
	msg, err := processMessage(msg)
	if err != nil {
		return err
	}

	client := initialization.GetLarkClient()

	// 构建 Markdown 格式消息内容
	content := fmt.Sprintf(`{
        "text": "%s",
        "tag": "markdown"
    }`, escapeJSONString(msg)) // 注意转义特殊字符

	// 使用 MsgType = "markdown" 而不是 "text"
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType("markdown"). // 关键修改：指定消息类型为 markdown
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	if err != nil {
		fmt.Println("API Error:", err)
		return err
	}

	if !resp.Success() {
		fmt.Println("Server Error:", resp.Code, resp.Msg, resp.RequestId())
		return fmt.Errorf("server error: %s", resp.Msg)
	}
	return nil
}

// 辅助函数：转义字符串中的特殊字符（如引号、换行符等）
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func uploadImage(base64Str string) (*string, error) {
	imageBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	client := initialization.GetLarkClient()
	resp, err := client.Im.Image.Create(context.Background(),
		larkim.NewCreateImageReqBuilder().
			Body(larkim.NewCreateImageReqBodyBuilder().
				ImageType(larkim.ImageTypeMessage).
				Image(bytes.NewReader(imageBytes)).
				Build()).
			Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, err
	}
	return resp.Data.ImageKey, nil
}
func replyImage(ctx context.Context, ImageKey *string,
	msgId *string) error {
	//fmt.Println("sendMsg", ImageKey, msgId)

	msgImage := larkim.MessageImage{ImageKey: *ImageKey}
	content, err := msgImage.String()
	if err != nil {
		fmt.Println(err)
		return err
	}
	client := initialization.GetLarkClient()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeImage).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return err
	}
	return nil

}

func replayImageCardByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string, question string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = sendImageCard(ctx, *imageKey, msgId, sessionId, question)
	if err != nil {
		return err
	}
	return nil
}

func replayImagePlainByBase64(ctx context.Context, base64Str string,
	msgId *string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = replyImage(ctx, imageKey, msgId)
	if err != nil {
		return err
	}
	return nil
}

func replayVariantImageByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = sendVarImageCard(ctx, *imageKey, msgId, sessionId)
	if err != nil {
		return err
	}
	return nil
}

func sendMsg(ctx context.Context, msg string, chatId *string) error {
	//fmt.Println("sendMsg", msg, chatId)
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := initialization.GetLarkClient()
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	//fmt.Println("content", content)

	resp, err := client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId(*chatId).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return err
	}
	return nil
}
func sendClearCacheCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🆑 机器人提醒", larkcard.TemplateBlue),
		withMainMd("您确定要清除对话上下文吗？"),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"),
		withClearDoubleCheckBtn(sessionId))
	replyCard(
		ctx,
		msgId,
		newCard,
	)
}

func sendSystemInstructionCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("🥷  已进入角色扮演模式", larkcard.TemplateBlue),
		//withMainText(content),
		withMainMd(content),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"))
	replyCard(
		ctx,
		msgId,
		newCard,
	)
}

func sendPicCreateInstructionCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🖼️ 已进入图片创作模式", larkcard.TemplateBlue),
		withPicResolutionBtn(sessionId),
		withNote("提醒：回复文本或图片，让AI生成相关的图片。"))
	replyCard(
		ctx,
		msgId,
		newCard,
	)
}

func sendPicModeCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🖼️ 机器人提醒", larkcard.TemplateBlue),
		withMainMd("收到图片，是否进入图片创作模式？"),
		withNote("请注意，这将开始一个全新的对话，您将无法利用之前话题的历史信息"),
		withPicModeDoubleCheckBtn(sessionId))
	replyCard(
		ctx,
		msgId,
		newCard,
	)
}

func sendNewTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader("👻️ 已开启新的话题", larkcard.TemplateBlue),
		//withMainText(content),
		withMainMd(content),
		withNote("提醒：点击对话框参与回复，可保持话题连贯"))
	replyCard(
		ctx,
		msgId,
		newCard,
	)
}

func sendHelpCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader("🎒需要帮助吗？", larkcard.TemplateBlue),
		withMainMd("**我是小飞机，一款基于AI技术的智能聊天机器人！**"),
		withSplitLine(),
		withMdAndExtraBtn(
			"** 🆑 清除话题上下文**\n文本回复 *清除* 或 */clear*",
			newBtn("立刻清除", map[string]interface{}{
				"value":     "1",
				"kind":      ClearCardKind,
				"chatType":  UserChatType,
				"sessionId": *sessionId,
			}, larkcard.MessageCardButtonTypeDanger)),
		withSplitLine(),
		withMainMd("🥷 **角色扮演模式**\n文本回复*角色扮演* 或 */system*+空格+角色信息"),
		withSplitLine(),
		withMainMd("🎤 **AI语音对话**\n私聊模式下直接发送语音"),
		withSplitLine(),
		withMainMd("🎨 **图片创作模式**\n回复*图片创作* 或 */picture*"),
		withSplitLine(),
		withMainMd("👨‍💼 **常用角色管理** 🚧\n"+
			" 文本回复 *角色管理* 或 */manage*"),
		withSplitLine(),
		withMainMd("🔃️ **历史话题回档** 🚧\n"+
			" 进入话题的回复详情页,文本回复 *恢复* 或 */reload*"),
		withSplitLine(),
		withMainMd("📤 **话题内容导出** 🚧\n"+
			" 文本回复 *导出* 或 */export*"),
		withSplitLine(),
		withMainMd("🎰 **连续对话与多话题模式**\n"+
			" 点击对话框参与回复，可保持话题连贯。同时，单独提问即可开启全新新话题"),
		withSplitLine(),
		withMainMd("🎒 **需要更多帮助**\n文本回复 *帮助* 或 */help*"),
	)
	replyCard(
		ctx,
		msgId,
		newCard,
	)
}

func sendImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string, question string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn("再来一张", map[string]interface{}{
			"value":     question,
			"kind":      PicTextMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	replyCard(
		ctx,
		msgId,
		newCard,
	)
	return nil
}

func sendVarImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn("再来一张", map[string]interface{}{
			"value":     imageKey,
			"kind":      PicVarMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	replyCard(
		ctx,
		msgId,
		newCard,
	)
	return nil
}

func convertMarkdownToLarkPostJson(markdown string) (string, error) {
	lines := strings.Split(markdown, "\n")
	var content [][]map[string]interface{}
	var paragraph []map[string]interface{}
	inCode := false
	var codeBlock []string

	flushParagraph := func() {
		if len(paragraph) > 0 {
			content = append(content, paragraph)
			paragraph = nil
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if !inCode {
				flushParagraph()
				inCode = true
				codeBlock = nil
			} else {
				inCode = false
				flushParagraph()
				content = append(content, []map[string]interface{}{
					{"tag": "code", "text": strings.Join(codeBlock, "\n")},
				})
			}
			continue
		}

		if inCode {
			codeBlock = append(codeBlock, line)
			continue
		}

		if trimmed == "" {
			flushParagraph()
			continue
		}

		// 标题处理
		var level string
		switch {
		case strings.HasPrefix(trimmed, "# "):
			level = "heading1"
			trimmed = strings.TrimPrefix(trimmed, "# ")
		case strings.HasPrefix(trimmed, "## "):
			level = "heading2"
			trimmed = strings.TrimPrefix(trimmed, "## ")
		case strings.HasPrefix(trimmed, "### "):
			level = "heading3"
			trimmed = strings.TrimPrefix(trimmed, "### ")
		case strings.HasPrefix(trimmed, "#### "):
			level = "heading4"
			trimmed = strings.TrimPrefix(trimmed, "#### ")
		}
		if level != "" {
			flushParagraph()
			content = append(content, []map[string]interface{}{
				{"tag": level, "text": trimmed},
			})
			continue
		}

		// 列表项
		if strings.HasPrefix(trimmed, "- ") {
			trimmed = "• " + strings.TrimPrefix(trimmed, "- ")
		}

		paragraph = append(paragraph, map[string]interface{}{
			"tag":  "text",
			"text": trimmed,
		})
	}
	flushParagraph()

	data := map[string]interface{}{
		"zh_cn": map[string]interface{}{
			"title":   "AI 回复",
			"content": content,
		},
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
