package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"math"
	"sort"
	"time"

	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Threshold = 0.9
var KSimilar = 3

func isSimilar(up1, down1, up2, down2 float64) bool {
	v := math.Min(up1, up2) - math.Max(down1, down2)
	u := math.Max(up1, up2) - math.Min(down1, down2)
	if v / u >= Threshold {
		return true
	}
	return false
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("r.Host", r.Host)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	defer conn.Close()

	testing := false
	bytenum := 0

	for {
		_, cr, err := conn.NextReader()
		if err != nil {
			log.Println("Error during message reading:", err)
			break
		}

		buf := make([]byte, 1024) // 1k

		for {
			n, err := cr.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Println("read error:", err)
				}
				break
			}
			bytenum += n

			if testing == false && string(buf[:5]) == "start" {
				log.Println("Speed test start")
				testing = true
				sizeRecord := make([]int, 0)
				speedRecord := make([]float64, 0)
				go func() {
					for testing {
						time.Sleep(50*time.Millisecond) // 50ms
						sizeRecord = append(sizeRecord, bytenum)
						if len(sizeRecord) >= 2 {
							speedRecord = append(speedRecord, float64(sizeRecord[len(sizeRecord)-1]-sizeRecord[len(sizeRecord)-2])/0.05) // bytes per second
						}
					}
				}()
				go func() {
					time.Sleep(10*time.Second)
					testing = false
					conn.Close()
				}()
				go func() {
					CISSpeed := 0.0
					similarCnt := 0
					lastUp := 0.0
					lastDown := 0.0
					quan := make([]float64, 200)
					for i:=0;i<200;i++ {
						quan[i] = 1
					}
					for testing {
						time.Sleep(200*time.Millisecond) // 200ms

						id := make([]int, len(speedRecord))
						for j:=0;j<len(speedRecord);j++ {
							id[j] = j
						}
						sort.Slice(id, func(i, j int) bool {
							if speedRecord[id[i]] < speedRecord[id[j]] {
								return true
							}
							return false
						})

						bias := 0
						for ;bias<len(speedRecord) && speedRecord[id[bias]]==0; {
							bias++
						}
						n := len(speedRecord) - bias
						if n <= 20 {
							continue
						}

						up := 0.0
						down := 0.0
						k2l := 0.0
						minInterval := (speedRecord[id[n - 1 + bias]] - speedRecord[id[0 + bias]]) / float64(n - 1)
						for i := 0; i < n; i++ {
							qsum := quan[id[i + bias]]
							for j := i+1; j < n; j++ {
								qsum += quan[id[j + bias]]
								k2ltemp := qsum * qsum / math.Max(speedRecord[id[j + bias]] - speedRecord[id[i + bias]], minInterval)
								if k2ltemp > k2l {
									k2l = k2ltemp
									up = speedRecord[id[j + bias]]
									down = speedRecord[id[i + bias]]
								}
							}
						}
						res := 0.0
						cnt := 0
						CISSpeed = res / float64(cnt)
						for i := 0; i < n; i++ {
							if speedRecord[id[i + bias]] >= down && speedRecord[id[i + bias]] <= up {
								quan[id[i+bias]] *= 1.1
								res += speedRecord[id[i + bias]]
								cnt++
							}
						}
						CISSpeed = res / float64(cnt)
						log.Println("up", up, "down", down)
						log.Println("CISSpeed", CISSpeed)
						if isSimilar(up, down, lastUp, lastDown) {
							similarCnt++
							if similarCnt >= KSimilar {
								err = conn.WriteMessage(1, []byte("Result:" + fmt.Sprintf("%.8v,%.8v", CISSpeed/1024/1024*8, float64(bytenum)/1024/1024)))
								if err != nil {
									log.Println("Error during message writing:", err)
								}
								testing = false
								log.Println("Speed test finish")
							}
						} else {
							similarCnt = 0
						}
						lastDown = down
						lastUp = up
					}
				}()
			}
		}
	}
}

func main() {
	http.HandleFunc("/", socketHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
