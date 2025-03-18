package steps

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlaudaDevops/bdd/logger"
	"github.com/cucumber/godog"
	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AnalysisParams struct {
	Host       string `yaml:"host" json:"host"`
	User       string `yaml:"user" json:"user"`
	Pwd        string `yaml:"pwd" json:"pwd"`
	Token      string `yaml:"token" json:"token"`
	Component  string `yaml:"component" json:"component"`
	Branch     string `yaml:"branch" json:"branch"`
	MaxRetries int    `yaml:"maxRetries" json:"maxRetries"`
	Sleep      int    `yaml:"sleep" json:"sleep"`
}

func WaitForAnalysis(ctx context.Context, params *godog.DocString) (context.Context, error) {
	log := logger.LoggerFromContext(ctx)
	input := AnalysisParams{}
	if err := yaml.Unmarshal([]byte(params.Content), &input); err != nil {
		log.Error("failed to unmarshal analysis params", zap.Error(err))
		return ctx, err
	}

	input = defaultParams(input)

	var err error
	input.Token, err = getToken(ctx, input)
	if err != nil {
		log.Error("failed to get token", zap.Error(err))
		return ctx, err
	}

	ctx, err = waitAnalysis(ctx, input)
	if err != nil {
		log.Error("failed to wait analysis", zap.Error(err))
		return ctx, err
	}

	return ctx, nil
}

func waitAnalysis(ctx context.Context, params AnalysisParams) (context.Context, error) {
	log := logger.LoggerFromContext(ctx)
	url := fmt.Sprintf("%s/api/ce/activity?component=%s&type=REPORT&branch=%s", params.Host, params.Component, params.Branch)
	request := getRestfulClient(ctx).NewRequest()
	request.SetHeader("Authorization", fmt.Sprintf("Bearer %s", params.Token))

	log.Debug("wait analysis", zap.String("url", url))
	for i := 0; i < params.MaxRetries; i++ {
		result := struct {
			Tasks []struct {
				Status string `json:"status"`
			} `json:"tasks"`
		}{}

		request.SetResult(&result)
		resp, err := request.Get(url)
		if err != nil {
			log.Error("failed to check analysis result", zap.Error(err), zap.String("response", resp.String()))
			return ctx, err
		}

		success := true
		if len(result.Tasks) == 0 {
			success = false
		}

		for _, task := range result.Tasks {
			if task.Status != "SUCCESS" {
				success = false
				break
			}
		}

		if success {
			return ctx, nil
		}

		if i == params.MaxRetries-1 {
			log.Info("analysis failed, and max retries reached", zap.String("response", resp.String()))
		}

		log.Info("analysis is running, waiting...", zap.Int("sleep", params.Sleep))
		time.Sleep(time.Duration(params.Sleep) * time.Second)
	}
	return ctx, fmt.Errorf("analysis failed, and max retries reached")
}

func defaultParams(params AnalysisParams) AnalysisParams {
	if params.MaxRetries == 0 {
		params.MaxRetries = 20
	}

	if params.Sleep == 0 {
		params.Sleep = 5
	}

	if params.Branch == "" {
		params.Branch = "main"
	}

	return params
}

var (
	restfulClientInstance *resty.Client
	once                  sync.Once
)

func getRestfulClient(ctx context.Context) *resty.Client {
	once.Do(func() {
		client := resty.New()
		restfulClientInstance = client
	})
	return restfulClientInstance
}

func getToken(ctx context.Context, params AnalysisParams) (string, error) {
	log := logger.LoggerFromContext(ctx)
	log.Info("get token", zap.String("host", params.Host), zap.String("user", params.User), zap.String("pwd", params.Pwd))
	if params.Token != "" {
		return params.Token, nil
	}

	req := getRestfulClient(ctx).NewRequest().SetBasicAuth(params.User, params.Pwd)
	result := struct {
		Token string `json:"token"`
	}{}
	req.SetResult(&result)
	url := fmt.Sprintf("%s/api/user_tokens/generate?name=my-token-%s", params.Host, time.Now().Format("20060102150405"))
	log.Debug("get token", zap.String("url", url))
	resp, err := req.Post(url)
	if err != nil {
		log.Error("failed to get token", zap.Error(err), zap.String("response", resp.String()))
		return "", err
	}
	log.Info("token", zap.String("token", result.Token))
	return result.Token, nil
}
