// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2020 Renesas Inc.
// Copyright 2020 EPAM Systems Inc.
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

package usersadapter_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"aos_vis/plugins/usersadapter"
)

/*******************************************************************************
 * Consts
 ******************************************************************************/

const usersVISPath = "Attribute.Vehicle.VehicleIdentification.Users"

/*******************************************************************************
 * Vars
 ******************************************************************************/

var tmpDir string

/*******************************************************************************
 * Init
 ******************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

/*******************************************************************************
 * Main
 ******************************************************************************/

func TestMain(m *testing.M) {
	var err error

	tmpDir, err = ioutil.TempDir("", "vis_")
	if err != nil {
		log.Fatalf("Error creating tmp dir: %s", err)
	}

	ret := m.Run()

	if err := os.RemoveAll(tmpDir); err != nil {
		log.Fatalf("Error removing tmp dir: %s", err)
	}

	os.Exit(ret)
}

/*******************************************************************************
 * Tests
 ******************************************************************************/

func TestGetName(t *testing.T) {
	adapter, err := usersadapter.New(generateConfig(usersVISPath, path.Join(tmpDir, "user.txt")))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	name := adapter.GetName()

	if name != "usersadapter" {
		t.Errorf("Wrong adapter name: %s", name)
	}
}

func TestEmptyUser(t *testing.T) {
	userFile := path.Join(tmpDir, "users.txt")
	if err := os.RemoveAll(userFile); err != nil {
		t.Fatalf("Can't remove Users file: %s", err)
	}

	adapter, err := usersadapter.New(generateConfig(usersVISPath, userFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{usersVISPath})

	if _, ok := data[usersVISPath]; !ok {
		t.Fatal("User not found in data")
	}

	users, ok := data[usersVISPath].([]string)
	if !ok {
		t.Fatal("Wrong Users data type")
	}

	if !reflect.DeepEqual(users, []string{}) {
		t.Errorf("Wrong Users value: %s", users)
	}
}

func TestExistingUser(t *testing.T) {
	userFile := path.Join(tmpDir, "users.txt")
	originUsers := []string{"claim0", "claim1", "claim2"}

	if err := writeUsers(userFile, originUsers); err != nil {
		t.Fatalf("Can't create users file: %s", err)
	}

	adapter, err := usersadapter.New(generateConfig(usersVISPath, userFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	data, err := adapter.GetData([]string{usersVISPath})

	if _, ok := data[usersVISPath]; !ok {
		t.Fatal("Users not found in data")
	}

	users, ok := data[usersVISPath].([]string)
	if !ok {
		t.Fatal("Wrong Users data type")
	}

	if !reflect.DeepEqual(originUsers, users) {
		t.Errorf("Wrong Users value: %s", users)
	}
}

func TestSetUser(t *testing.T) {
	usersFile := path.Join(tmpDir, "users.txt")
	if err := os.RemoveAll(usersFile); err != nil {
		t.Fatalf("Can't remove Users file: %s", err)
	}

	adapter, err := usersadapter.New(generateConfig(usersVISPath, usersFile))
	if err != nil {
		t.Fatalf("Can't create adapter: %s", err)
	}
	defer adapter.Close()

	setUsers := []string{"claim0", "claim1", "claim2"}

	if err = adapter.Subscribe([]string{usersVISPath}); err != nil {
		t.Fatalf("Subscribe error: %s", err)
	}

	if err = adapter.SetData(map[string]interface{}{usersVISPath: setUsers}); err != nil {
		t.Fatalf("Set data error: %s", err)
	}

	select {
	case data := <-adapter.GetSubscribeChannel():
		if !reflect.DeepEqual(data[usersVISPath], setUsers) {
			t.Errorf("Wrong Users value: %s", setUsers)
		}

	case <-time.After(5 * time.Second):
		t.Error("Wait data change timeout")
	}
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func generateConfig(visPath, filePath string) (config []byte) {
	type adapterConfig struct {
		VISPath  string `json:"VISPath"`
		FilePath string `json:"FilePath"`
	}

	var err error

	if config, err = json.Marshal(&adapterConfig{VISPath: visPath, FilePath: filePath}); err != nil {
		log.Fatalf("Can't marshal config: %s", err)
	}

	return config
}

func writeUsers(usersFile string, users []string) (err error) {
	file, err := os.Create(usersFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, claim := range users {
		fmt.Fprintln(writer, claim)
	}

	return writer.Flush()
}
