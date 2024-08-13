package service

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"sync"

	"log"
	"net/http"
)

// 用户发送
type SendMsg struct {
	Type    int    `json:"type"`
	Content string `json:"content"`
}

// 用户回复
type ReplyMsg struct {
	From    string `json:"from"`
	Code    int    `json:"code"`
	Content string
}

// 返回给用户的历史消息
type HistoryMsg struct {
	Msgs  []Message `json:"msgs"`
	Total int       `json:"total"`
	Type  int       `json:"type"`
}

// 用户实例
type Client struct {
	ID     string
	SendID string
	Socket *websocket.Conn
	Send   chan []byte
}

// 广播
type Broadcast struct {
	Client  *Client
	Message []byte
	Type    int
}

// 用户管理
type ClientManager struct {
	Clients    map[string]*Client //用户管理
	Broadcast  chan *Broadcast    //广播通道
	Register   chan *Client       //用户注册通道
	Unregister chan *Client       //用户注销通道
}

type Message struct {
	Sender  string `json:"sender,omitempty"`  //发送人
	Content string `json:"content,omitempty"` //内容
}
type HistoryStruct struct {
	msgMutex sync.RWMutex //读写锁
	Msgs     []Message    `json:"msgs"` //历史消息
}

var HMsg = &HistoryStruct{Msgs: make([]Message, 0), msgMutex: sync.RWMutex{}}
var Manager = &ClientManager{
	Clients:   make(map[string]*Client),
	Broadcast: make(chan *Broadcast),
	//用通道实现

	Register:   make(chan *Client),
	Unregister: make(chan *Client),
}

func ChatHandler(ctx *gin.Context) {

	uid := ctx.Query("uid")

	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		}}).Upgrade(ctx.Writer, ctx.Request, nil) //升级ws协议
	if err != nil {
		log.Println("err:", err)
		http.NotFound(ctx.Writer, ctx.Request)
		return
	}
	println("uid:", uid)

	//创建用户实例
	client := &Client{
		ID:     uid,
		Socket: conn,
		Send:   make(chan []byte),
	}
	//用户注册到用户管理
	Manager.Register <- client
	go client.Read()
	go client.Write()
}

// 读取用户传入
func (c *Client) Read() {
	defer func() {
		Manager.Unregister <- c
		_ = c.Socket.Close()
	}()

	for {

		c.Socket.PongHandler()
		sendMsg := new(SendMsg)
		//c.Socket.ReadMessage()
		err := c.Socket.ReadJSON(&sendMsg)
		if err != nil {
			log.Println("数据格式不正确", err)
			Manager.Unregister <- c
			c.Socket.Close()
			break
		}
		fmt.Println("sendMsg:", sendMsg)
		if sendMsg.Type == 1 {
			HMsg.msgMutex.Lock()

			HMsg.Msgs = append(HMsg.Msgs, Message{
				Sender:  c.ID,
				Content: fmt.Sprintf("%s", sendMsg.Content),
			})
			HMsg.msgMutex.Unlock()
			replyMsg := ReplyMsg{
				From:    c.ID,
				Code:    0,
				Content: sendMsg.Content,
			}
			data, _ := json.Marshal(replyMsg)
			Manager.Broadcast <- &Broadcast{
				Client:  c,
				Message: data, //发送过来的消息
			}

		} else if sendMsg.Type == 2 {

			results := History(10) //获取历史消息十条
			fmt.Println("id:", c.SendID, c.ID)
			if len(results) > 10 {
				results = results[10:]
			} else if len(results) == 0 {
				replyMsg := ReplyMsg{

					Code:    -1,
					Content: "上面没有消息了",
				}
				msg, _ := json.Marshal(replyMsg)
				_ = c.Socket.WriteMessage(websocket.TextMessage, msg)
				continue
			}
			replyMsg := HistoryMsg{
				Msgs:  results,
				Total: len(results),
				Type:  2,
			}

			msg, _ := json.Marshal(replyMsg)
			_ = c.Socket.WriteMessage(websocket.TextMessage, msg)
		}

	}
}
func (c *Client) Write() {
	defer func() {
		_ = c.Socket.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				_ = c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			replyMsg := ReplyMsg{

				Code:    200,
				Content: fmt.Sprintf("%s", string(message)),
			}

			msg, _ := json.Marshal(replyMsg)
			_ = c.Socket.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

func History(n int) []Message {
	//读取n条历史消息
	HMsg.msgMutex.RLock()
	defer HMsg.msgMutex.RUnlock()
	if len(HMsg.Msgs) < n {
		return HMsg.Msgs
	}
	return HMsg.Msgs[len(HMsg.Msgs)-n:]
}
