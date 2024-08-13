package service

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
)

func (manager *ClientManager) HandleWebSocketEvents() {
	for { //循环监听
		select {
		case conn := <-Manager.Register: //将连接放入用户管理
			fmt.Printf("有新连接 %v\n ", conn.ID)
			Manager.Clients[conn.ID] = conn
			replyMsg := ReplyMsg{

				Code:    200,
				Content: "已经连接到服务器",
			}
			msg, _ := json.Marshal(replyMsg)
			_ = conn.Socket.WriteMessage(websocket.TextMessage, msg)
		case conn := <-Manager.Unregister: //删除连接
			fmt.Printf("注销连接:%s", conn.ID)
			if _, ok := Manager.Clients[conn.ID]; ok { //若该连接以注册
				replyMsg := ReplyMsg{

					Code:    -1,
					Content: "连接中断",
				}
				msg, _ := json.Marshal(replyMsg)
				_ = conn.Socket.WriteMessage(websocket.TextMessage, msg)
				close(conn.Send)
				delete(Manager.Clients, conn.ID)
			}

			//有人发消息
		case brodcast := <-Manager.Broadcast:
			data := brodcast.Message

			//Manager.Clients是用户连接表
			for _, conn := range Manager.Clients {

				conn.Socket.WriteMessage(websocket.TextMessage, data)
			}

		}
	}
}
