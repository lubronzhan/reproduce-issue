package localdirector

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Client struct {
	Host string
	Port int
}

type FactoryConfig struct {
	Host string
	Port int

	// CA certificate is not required
	CACert string

	Client       string
	ClientSecret string

	TokenFunc func(bool) (string, error)
}

func (c *Client) directorConfig() (director.FactoryConfig, error) {
	return director.FactoryConfig{
		Host: "10.92.214.87",
		Port: 25555,
	}, nil
}

type BoshHTTP struct {
	client *Client
}

type HTTP interface {
	RawGet(path string) (string, error)
	RawPost(path, data, contentType string) (string, error)
	RawDelete(path string) (string, error)
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_http_factory.go . HTTPFactory

func NewBoshHTTP(client *Client) HTTP {
	return &BoshHTTP{
		client: client,
	}
}

func (b *BoshHTTP) RawGet(path string) (string, error) {
	fileReporter := director.NewNoopFileReporter()
	logger := boshlog.NewLogger(boshlog.LevelError)
	config, err := b.client.directorConfig()
	if err != nil {
		return "", nil
	}

	hc, err := b.httpClient(config, logger)
	if err != nil {
		return "", err
	}

	log.Println(fmt.Sprintf("https://%s:%d", config.Host, config.Port))
	cr := director.NewClientRequest(fmt.Sprintf("https://%s:%d", config.Host, config.Port), hc, fileReporter, logger)
	w := bytes.NewBuffer([]byte{})

	log.Println("Sending request")
	_, _, err = cr.RawGet(path, w, nil)
	if err != nil {
		fmt.Println("error?")
		return "", err
	}
	fmt.Println("response?")
	return string(w.Bytes()), nil
}

func (b *BoshHTTP) RawPost(path, data, contentType string) (string, error) {
	fileReporter := director.NewNoopFileReporter()
	logger := boshlog.NewLogger(boshlog.LevelError)
	config, err := b.client.directorConfig()
	if err != nil {
		return "", nil
	}

	hc, err := b.httpClient(config, logger)
	if err != nil {
		return "", err
	}

	cr := director.NewClientRequest(fmt.Sprintf("https://%s:%d", config.Host, config.Port), hc, fileReporter, logger)

	var contentTypeWrapper func(*http.Request)
	if contentType != "" {
		contentTypeWrapper = func(req *http.Request) {
			req.Header.Add("Content-Type", contentType)
		}
	}
	w, _, err := cr.RawPost(path, []byte(data), contentTypeWrapper)
	if err != nil {
		return "", err
	}
	return string(w), nil
}

func (b *BoshHTTP) RawDelete(path string) (string, error) {
	fileReporter := director.NewNoopFileReporter()
	logger := boshlog.NewLogger(boshlog.LevelError)
	config, err := b.client.directorConfig()
	if err != nil {
		return "", nil
	}

	hc, err := b.httpClient(config, logger)
	if err != nil {
		return "", err
	}

	cr := director.NewClientRequest(fmt.Sprintf("https://%s:%d", config.Host, config.Port), hc, fileReporter, logger)
	r, _, err := cr.RawDelete(path)
	if err != nil {
		return "", err
	}
	return string(r), nil
}

func (b *BoshHTTP) httpClient(config director.FactoryConfig, logger boshlog.Logger) (*httpclient.HTTPClient, error) {
	certPool, err := config.CACertPool()
	if err != nil {
		return nil, err
	}

	rawClient := httpclient.CreateDefaultClient(certPool)
	authAdjustment := director.NewAuthRequestAdjustment(
		nil, "a", "s")
	rawClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {

		// Since redirected requests are not retried,
		// forcefully adjust auth token as this is the last chance.
		err := authAdjustment.Adjust(req, true)
		if err != nil {
			return err
		}

		req.URL.Host = net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port))
		return nil
	}

	retryClient := httpclient.NewNetworkSafeRetryClient(rawClient, 5, 500*time.Millisecond, logger)

	authedClient := director.NewAdjustableClient(retryClient, authAdjustment)

	httpOpts := httpclient.Opts{NoRedactUrlQuery: true}
	httpClient := httpclient.NewHTTPClientOpts(authedClient, logger, httpOpts)

	return httpClient, nil
}
