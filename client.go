package amocrm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type (
	// Информация о подключении к аккаунту
	clientInfo struct {
		userLogin string
		apiHash   string
		Timezone  string
		Url       string
		Cookie    []*http.Cookie
	}
	//AuthResponse Структура ответа при авторизации
	AuthResponse struct {
		Response struct {
			Auth     bool `json:"auth"`
			Accounts []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Subdomain string `json:"subdomain"`
				Language  string `json:"language"`
				Timezone  string `json:"timezone"`
			} `json:"accounts"`
			ServerTime int    `json:"server_time"`
			Error      string `json:"error"`
		} `json:"response"`
	}
	//respID стандартный ответ
	respID struct {
		Embedded struct {
			Items []struct {
				ID int `json:"id"`
			} `json:"items"`
		} `json:"_embedded"`
		Response struct {
			Error string `json:"error"`
		} `json:"response"`
	}
)

//New Открытия соединения и авторизация
func New(accountURL string, login string, hash string) (*clientInfo, error) {
	var err error

	if login == "" {
		return nil, errors.New("login is empty")
	}
	if hash == "" {
		return nil, errors.New("hash is empty")
	}
	c := &clientInfo{
		userLogin: login,
		apiHash:   hash,
	}
	_, err = url.Parse(accountURL)
	if err != nil {
		return nil, err
	}
	c.Url = accountURL
	values := url.Values{}
	values.Set("USER_LOGIN", c.userLogin)
	values.Set("USER_HASH", c.apiHash)
	reqbody := strings.NewReader(values.Encode())
	urlString := c.Url + apiUrls["auth"]
	resp, err := http.Post(urlString, "application/x-www-form-urlencoded", reqbody)

	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		c.Cookie = resp.Cookies()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var authResponse AuthResponse
		err = json.Unmarshal(body, &authResponse)

		if err != nil {
			return nil, err
		}
		if len(authResponse.Response.Accounts) > 0 {
			c.Timezone = authResponse.Response.Accounts[0].Timezone
		}
		if !authResponse.Response.Auth {
			return nil, errors.New(authResponse.Response.Error)
		}
		return c, nil
	} else {
		err = errors.New("Wrong http status: " + string(resp.StatusCode))
		return nil, err
	}
}

func (c *clientInfo) DoGet(url string, data map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for _, cookie := range c.Cookie {
		req.AddCookie(cookie)
	}
	q := req.URL.Query()
	for key, value := range data {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *clientInfo) DoPost(url string, data interface{}) (*http.Response, error) {
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	fmt.Println(url)
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range c.Cookie {
		req.AddCookie(cookie)
	}
	fmt.Println(req)
	client := &http.Client{}
	return client.Do(req)
}

func (c *clientInfo) DoPostWithoutCookie(url string, data interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, err
	}
	fmt.Println(&buf)
	// jsonStr := "add%5B0%5D%5Bsource_name%5D=686&add%5B0%5D%5Bsource_uid%5D=1544023966.361&add%5B0%5D%5Bcreated_at%5D=1544040279&add%5B0%5D%5Bincoming_entities%5D%5Bleads%5D%5B0%5D%5Bname%5D=89626849787&add%5B0%5D%5Bincoming_lead_info%5D%5Bto%5D=78123375496&add%5B0%5D%5Bincoming_lead_info%5D%5Bfrom%5D=89626849787&add%5B0%5D%5Bincoming_lead_info%5D%5Bdate_call%5D=1544040279&add%5B0%5D%5Bincoming_lead_info%5D%5Bduration%5D=14&add%5B0%5D%5Bincoming_lead_info%5D%5Blink%5D=https%3A%2F%2Fnewcity.dela.bz%3A8085%2Fgetrecord%3Fpath%3D2018-12-05%2F18-32-54-78123375496-89626849788.mp3&add%5B0%5D%5Bincoming_lead_info%5D%5Bservice_code%5D=amo_asterisk&add%5B0%5D%5Bincoming_lead_info%5D%5Buniq%5D=1544023966.361&add%5B0%5D%5Bincoming_lead_info%5D%5Badd_note%5D=1"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fmt.Println(req)
	client := &http.Client{}
	return client.Do(req)
}

func (c *clientInfo) GetResponseID(resp *http.Response) (int, error) {
	result := respID{}
	dec := json.NewDecoder(resp.Body)
	err := dec.Decode(&result)
	if err != nil {
		return 0, err
	}
	if len(result.Embedded.Items) == 0 {
		if result.Response.Error != "" {
			return 0, errors.New(result.Response.Error)
		}
		return 0, errors.New("No Items")
	}
	return result.Embedded.Items[0].ID, nil
}
