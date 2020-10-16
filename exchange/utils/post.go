package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

func Verify(publickey string) {
	url_regular := "http://localhost:1423/verify"

	data := make(url.Values)
	data["publicKey"] = []string{publickey}

	resp, err := http.PostForm(url_regular, data)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	//var jsonMap map[string]interface{}
	//err = json.Unmarshal(bodyC, &jsonMap)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	fmt.Println(string(bodyC))
}

func sendTransaction() {

}