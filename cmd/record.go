// Copyright © 2016 Sidharth Kshatriya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"strconv"
	"github.com/spf13/cobra"
	"bytes"
	"log"
	"os/exec"
	"net"
)

var docroot string

// recordCmd represents the record command
var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "start the built in PHP server and record execution",
	Run: func(cmd *cobra.Command, args []string) {
		startBasicDebuggerClient()
		doRecordSession()
	},
}

func init() {
	RootCmd.AddCommand(recordCmd)
	recordCmd.Flags().StringVar(&docroot, "docroot", "", "server docroot")
}

func doRecordSession() {
	recordSession := exec.Command("rr", "record", "php", "-S", "127.0.0.1:8088", "-t", docroot)
	stderr, err := recordSession.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = recordSession.Start()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully started recording session... Press Ctrl-C to terminate recording")
	fmt.Println("PHP built in cli server is running at 127.0.0.1:8088 with docroot:", docroot)
	buf := make([]byte, 100)
	n, _ := stderr.Read(buf)
	for n > 0 {
		fmt.Print(string(buf[0:n]))
		n, _ = stderr.Read(buf)
	}

	err = recordSession.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func startBasicDebuggerClient() {
	listener, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Dontbug DBGp debugger client is listening on 127.0.0.1:9000 for connections from PHP")
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
			}

			go func(conn net.Conn) {
				buf := make([]byte, 2048)
				seq := 0
				for {
					bytesRead, _ := conn.Read(buf)
					if (bytesRead <= 0) {
						return
					}

					nullAt := bytes.IndexByte(buf, byte(0))
					if nullAt == -1 {
						log.Fatal("Could not find length in debugger engine response")
					}

					dataLen, err := strconv.Atoi(string(buf[0:nullAt]))
					if err != nil {
						log.Fatal(err)
					}

					bytesLeft := dataLen - (bytesRead - nullAt - 2)
					// fmt.Println("bytes_left:", bytes_left, "data_len:", data_len, "bytes_read:", bytes_read, "null_at:", null_at)
					if bytesLeft != 0 {
						log.Fatal("There are still some bytes left to receive. Strange")
					}

					fmt.Println("<-", string(buf[nullAt + 1:bytesRead - 1]))
					seq++

					// Keep running until we are able to record the execution
					runCommand := fmt.Sprintf("run -i %d\x00", seq)
					conn.Write([]byte(runCommand))
					fmt.Println("->", runCommand)
				}
			}(conn)
		}
	}()
}