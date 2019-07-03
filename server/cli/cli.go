package cli

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/hugowan/chat/pbx"
	"google.golang.org/grpc"
)

const (
	RootUserID   string = "usrH7YetPkQ-c4"
	RootUsername string = "xena"
	RootPassword string = "xena123"

	MasterUserID   string = "usrVT4nyfUlMs8"
	MasterUsername string = "85251966260"
	MasterTel      string = "85251966260"

	// DefaultPassword string = "XCf5mcQqqTJ4N8kx"
	DefaultPassword string = "000000"
)

var globals struct {
	conn   *grpc.ClientConn
	stream pbx.Node_MessageLoopClient
}
var wg sync.WaitGroup
var mu sync.Mutex

type pubData struct {
	Topic   string
	NoEcho  bool
	Head    map[string][]byte
	Content []byte
}

type MsgData struct {
	Id          string
	SenderTel   string
	ReceiverTel string
	Content     string
	MimeType    string
}

func main() {
	// fmt.Println(GetUserIdByTel("85251966260"))
	// fmt.Println(GetUserIdByTel("14155238886"))
	CreateUser("14155238886")
	return

	msgData := MsgData{
		Id:          "0000000000",
		SenderTel:   "1111111111",
		ReceiverTel: "2222222222",
		Content:     "test message",
		MimeType:    "",
	}
	SendMessage(msgData)

	return

	////////////////////////////////////////

	// 85251966260 --> 14155238886
	// msgData := MsgData{
	// 	Id:          "0000000000",
	// 	SenderTel:   "85251966260",
	// 	ReceiverTel: "14155238886",
	// 	Content:     "/tmp/whatsapp/15013D1FBA0EADBF8D39EF02FF0FCF39.jpeg",
	// 	MimeType:    "image/jpeg",
	// }
	// SendMessage(msgData)

	// 14155238886 --> 85251966260
	// msgData = MsgData{X
	// 	Id:          "0000000000",
	// 	SenderTel:   "14155238886",
	// 	ReceiverTel: "85251966260",
	// 	Content:     "/tmp/whatsapp/15013D1FBA0EADBF8D39EF02FF0FCF39.jpeg",
	// 	MimeType:    "image/jpeg",
	// }
	// SendMessage(msgData)

	/**************************************************/

	// parse gRPC response
	// go HiMsg(0000)
	// serverMsg, err = globals.stream.Recv()
	// fmt.Println(serverMsg.GetCtrl())
	// fmt.Println(string(serverMsg.GetCtrl().Id))
	// fmt.Println(string(serverMsg.GetCtrl().Params["build"]))
	// fmt.Println(string(serverMsg.GetCtrl().Params["ver"]))
	// return

	/**************************************************/

	/*
		waitc := make(chan struct{})
		go func() {
			for {
				serverMsg, err := globals.stream.Recv()
				if err == io.EOF {
					close(waitc)
					return
				}
				if err != nil {
					log.Fatal(err)
				}
				log.Println(serverMsg)
			}
		}()
		<-waitc
	*/
	/*
		serverMsg, err = globals.stream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		log.Println(serverMsg)
	*/
}

func Connect() {
	var err error
	globals.conn, err = grpc.Dial("localhost:6061", grpc.WithInsecure())
	if err != nil {
		log.Fatal("Error dialing", err)
	}
	// defer globals.conn.Close()

	client := pbx.NewNodeClient(globals.conn)

	// ctx, _ := context.WithTimeout(context.Background(), 3600*time.Second)
	// defer cancel()

	globals.stream, err = client.MessageLoop(context.Background())
	if err != nil {
		log.Fatal("Error calling", err)
	}
}

func Disconnect() {
	globals.stream.CloseSend()
}

func GetUserIdByTel(tel string) (string, bool) {
	Connect()
	id := genRandomNumber()

	HiMsg(id)
	serverMsg, _ := globals.stream.Recv()
	log.Println(serverMsg)

	// login as root
	id++
	LoginMsg(id, RootUsername, RootPassword)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	// subscribe fnd
	id++
	subData := make(map[string]interface{})
	subData["Topic"] = "fnd"
	SubMsg(id, subData)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	// fnd by whatsapp number
	id++
	setData := make(map[string]interface{})
	setData["Topic"] = "fnd"
	setData["Public"] = "\"whatsapp:" + tel + "\""
	SetMsg(id, setData)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	id++
	getData := make(map[string]interface{})
	getData["Topic"] = "fnd"
	getData["What"] = "sub"
	GetMsg(id, getData)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	Disconnect()

	if serverMsg.GetMeta() != nil {
		color.Green("User exists: %v\t%v", tel, serverMsg.GetMeta().Sub[0].UserId)
		return serverMsg.GetMeta().Sub[0].UserId, true
	} else {
		color.Red("User does not exist: %v", tel)
		return "", false
	}
}

// login no required
func CreateUser(tel string) {
	Connect()

	id := genRandomNumber()
	HiMsg(id)
	serverMsg, _ := globals.stream.Recv()
	log.Println(serverMsg)

	id++
	AccMsg(id, tel)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	Disconnect()
}

func SendMessage(m MsgData) {
	mu.Lock()

	_, senderExists := GetUserIdByTel(m.SenderTel)
	if senderExists == false {
		CreateUser(m.SenderTel)
	}

	// get receiver user id
	receiverUserID, receiverExists := GetUserIdByTel(m.ReceiverTel)
	if receiverExists == false {
		CreateUser(m.ReceiverTel)
		receiverUserID, _ = GetUserIdByTel(m.ReceiverTel)
	}

	/*** START TO SEND MESSAGE TO TINODE ***/
	// make new connection to make sure new login ready
	Connect()
	id := genRandomNumber()

	// handshake
	HiMsg(id)
	serverMsg, _ := globals.stream.Recv()
	log.Println(serverMsg)

	// login as sender
	id++
	LoginMsg(id, m.SenderTel, DefaultPassword)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	// subscribe to receiver
	id++
	subData := make(map[string]interface{})
	subData["Topic"] = receiverUserID
	SubMsg(id, subData)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	// publish to receiver
	head := make(map[string][]byte)
	head["origin"] = encodeToBytes("whatsapp")
	head["origin-message-id"] = encodeToBytes(m.Id)

	var byteContent []byte
	if m.MimeType != "" {
		// attachment
		head["mime"] = encodeToBytes("text/x-drafty")

		switch mime := strings.Split(m.MimeType, "/")[0]; mime {
		case "image":
			inlineImageContent := inlineImage(m)
			byteContent = encodeToBytes(inlineImageContent)
		case "audio":
			// TODO:
		}
	} else {
		// plain text
		byteContent = encodeToBytes(m.Content)
	}

	id++
	pubData := &pubData{Topic: receiverUserID, Head: head, Content: byteContent}
	PubMsg(id, pubData)
	serverMsg, _ = globals.stream.Recv()
	log.Println(serverMsg)

	Disconnect()
	mu.Unlock()
}

/**************************************************/

func HiMsg(id int) {
	hi := &pbx.ClientHi{}
	hi.Id = strconv.Itoa(id)
	hi.UserAgent = "Golang_Spider_Bot/3.0"
	hi.Ver = "0.15"
	hi.Lang = "EN"

	msgHi := &pbx.ClientMsg_Hi{hi}
	clientMessage := &pbx.ClientMsg{Message: msgHi}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

func AccMsg(id int, tel string) {
	// var cred []*pbx.Credential
	// cred = append(cred, &pbx.Credential{Method: "email", Value: "test@123.com"})
	// cred = append(cred, &pbx.Credential{Method: "email", Response: "test@123.com"})

	acc := &pbx.ClientAcc{}
	acc.Id = strconv.Itoa(id)
	acc.UserId = "new"
	acc.Scheme = "basic"
	acc.Secret = []byte(tel + ":" + DefaultPassword)
	acc.Login = false
	acc.Tags = []string{"whatsapp:" + tel + ""}
	acc.Desc = &pbx.SetDesc{DefaultAcs: &pbx.DefaultAcsMode{Auth: "", Anon: ""}, Public: []byte("{\"fn\": \"" + tel + "\"}")}
	// acc.Cred = cred

	msgAcc := &pbx.ClientMsg_Acc{acc}
	clientMessage := &pbx.ClientMsg{Message: msgAcc}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

func LoginMsg(id int, username string, password string) {
	color.Green("Logged in as: %v", username)

	login := &pbx.ClientLogin{}
	login.Id = strconv.Itoa(id)
	login.Scheme = "basic"
	login.Secret = []byte(username + ":" + password)

	msgLogin := &pbx.ClientMsg_Login{login}
	clientMessage := &pbx.ClientMsg{Message: msgLogin}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

func SubMsg(id int, data map[string]interface{}) {
	sub := &pbx.ClientSub{}
	sub.Id = strconv.Itoa(id)
	sub.Topic = data["Topic"].(string)
	sub.SetQuery = &pbx.SetQuery{Desc: &pbx.SetDesc{DefaultAcs: &pbx.DefaultAcsMode{}}, Sub: &pbx.SetSub{}}

	msgSub := &pbx.ClientMsg_Sub{sub}
	clientMessage := &pbx.ClientMsg{Message: msgSub}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

func PubMsg(id int, p *pubData) {
	pub := &pbx.ClientPub{}
	pub.Id = strconv.Itoa(id)
	pub.Topic = p.Topic
	pub.NoEcho = true
	pub.Head = p.Head
	pub.Content = p.Content

	msgPub := &pbx.ClientMsg_Pub{pub}
	clientMessage := &pbx.ClientMsg{Message: msgPub}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

func GetMsg(id int, data map[string]interface{}) {
	get := &pbx.ClientGet{}
	get.Id = strconv.Itoa(id)
	get.Topic = data["Topic"].(string)
	get.Query = &pbx.GetQuery{What: data["What"].(string)}

	msgGet := &pbx.ClientMsg_Get{get}
	clientMessage := &pbx.ClientMsg{Message: msgGet}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

func SetMsg(id int, data map[string]interface{}) {
	set := &pbx.ClientSet{}
	set.Id = strconv.Itoa(id)
	set.Topic = data["Topic"].(string)
	set.Query = &pbx.SetQuery{Desc: &pbx.SetDesc{DefaultAcs: &pbx.DefaultAcsMode{}, Public: []byte(data["Public"].(string))}, Sub: &pbx.SetSub{}}

	msgSet := &pbx.ClientMsg_Set{set}
	clientMessage := &pbx.ClientMsg{Message: msgSet}
	err := globals.stream.Send(clientMessage)
	if err != nil {
		log.Fatal("error sending message ", err)
	}
}

/**************************************************/

func encodeToBytes(src interface{}) []byte {
	b, _ := json.Marshal(src)
	return b
}

func genRandomNumber() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(900000)
}

func getEncodedImage(filepath string) string {
	file, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer file.Close()

	// read image into byte
	reader := bufio.NewReader(file)
	content, _ := ioutil.ReadAll(reader)

	// encode as base54
	encoded := base64.StdEncoding.EncodeToString(content)

	return encoded
}

func getImageDimension(filepath string) (int, int) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	im, _, err := image.DecodeConfig(file)

	return im.Width, im.Height
}

func inlineImage(m MsgData) map[string]interface{} {
	// fmt
	f := make(map[string]int)
	f["len"] = 1

	// ent
	width, height := getImageDimension(m.Content)

	entData := make(map[string]interface{})
	entData["height"] = height
	entData["mime"] = m.MimeType
	entData["name"] = path.Base(m.Content)
	entData["val"] = getEncodedImage(m.Content)
	entData["width"] = width

	e := make(map[string]interface{})
	e["data"] = entData
	e["tp"] = "IM"

	content := make(map[string]interface{})
	content["fmt"] = []map[string]int{0: f}
	content["txt"] = " "
	content["ent"] = []map[string]interface{}{0: e}

	return content
}
