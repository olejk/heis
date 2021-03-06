package network 

import (
	"fmt"
	"net"
	"encoding/json"
	"time"
	//"os"
)


const(

	elevatorDead = 1000000000
	HeartBeatPort = 30103
	StatusPort = 	30203


)


type Heartbeat struct {
		Id string
		Time time.Time
}

type Message struct {
	MessageType string //neworder,just arrived, status update, completed order,
	SenderIP    string
	elevators   map[string]Elevator
}

type Order struct {
	Direction int
	Floor     int
}

type Elevator struct {
	Direction       int
	LastPassedFloor int
	UpOrders        []bool
	DownOrders      []bool
	CommandOrders   []bool
}



func GetIP() string {

	addrs, error := net.InterfaceAddrs()
	if error != nil {	
    	fmt.Println("error:",error)
    	}

   	for _, address := range addrs {
    	if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
    		if ipnet.IP.To4() != nil {
            	return ipnet.IP.String()
    		}
		}
	}
	return ""
}




func UDPDial(port int) *net.UDPConn {

	casting,error:= net.ResolveUDPAddr("udp",fmt.Sprintf("129.241.187.255:%d", port)) 
	if error !=nil{
		fmt.Println("error:", error)	
	}
	socket,error:=net.DialUDP("udp",nil,casting)
	if error !=nil{
		fmt.Println("error:", error)
	}
	return socket
}


func UDPListen(port int) *net.UDPConn {
	local,error:= net.ResolveUDPAddr("udp",fmt.Sprintf(":%d", port)) 
	if error !=nil{
		fmt.Println("error:", error)		
	}


	socket,error:=net.ListenUDP("udp",local)
	if error !=nil{
		fmt.Println("error:", error)	
	}
	return socket
}


func UDPRx(rx chan []byte ,port int){
	
	socket := UDPListen(port)

	for{
		socket.SetReadDeadline(time.Now().Add(10*time.Second)) //ingen aktivitet på net i løpet av 10s, noe er feil ?=
		buffer := make([]byte,1024)
		n,_,error := socket.ReadFromUDP(buffer) 
		
		if error !=nil{
			fmt.Println("error:", error)		
		}
		
		buffer = buffer[:n]
		rx <- buffer
	}

}

func UDPTx(tx chan []byte,port int)  {
	
	socket := UDPDial(port)

	for{
		socket.SetWriteDeadline(time.Now().Add(10*time.Second))
		_,error := socket.Write(<- tx)
		if error !=nil{
			fmt.Println("error:", error)
		}
		time.Sleep(30*time.Millisecond)	
	}	
}


func SendHeartBeat(){
	send := make(chan []byte,1)
	go UDPTx(send,HeartBeatPort)

	for{
		myBeat := Heartbeat{GetIP(),time.Now()}
		myBeatBs,error := json.Marshal(myBeat)
		
		if error !=nil{
			fmt.Println("error:", error)
		}
	 	send <- myBeatBs
	 	time.Sleep(300*time.Millisecond)
	}
}

func HeartbeatTransceiver(newElevator chan string,deadElevator chan string) {
	
	receive := make(chan []byte,1)
	heartbeats := make(map[string]*time.Time)
	go UDPRx(receive,HeartBeatPort)
	go SendHeartBeat()

	for{	
	 	otherBeatBs := <-receive
	 	
	 	otherBeat := Heartbeat{}
	 	error := json.Unmarshal(otherBeatBs,&otherBeat)
		if error !=nil{
			fmt.Println("error:", error)
		}
		_,ok := heartbeats[otherBeat.Id]
		
		if ok {
			heartbeats[otherBeat.Id] =&otherBeat.Time
		}else{
			newElevator <- otherBeat.Id
			heartbeats[otherBeat.Id] =&otherBeat.Time 
			
		}

		for i,t := range heartbeats {
			dur := time.Since(*t)
			if dur.Seconds() > 1 {
				deadElevator <- i
				delete(heartbeats,i)
			}
		}
	}
}

func SendStatus(toPass chan Message){
	send := make(chan []byte)
	go UDPTx(send,StatusPort)
	
	for{

		toPassBs,error := json.Marshal(<-toPass)
		if error !=nil{
			fmt.Println("error:", error)
		}
		send<-toPassBs
	}

}


func StatusTransceiver(toPass chan Message,toGet chan Message){

	receive := make(chan []byte)
	
	go UDPRx(receive,StatusPort)
	go SendStatus(toPass)

	
	for{
		RxMessageBs:=<-receive
		RxMessage := Message{}
	 	error := json.Unmarshal(RxMessageBs,&RxMessage)
		if error !=nil{
			fmt.Println("error:", error)
		}
		toGet<-RxMessage
	}


}