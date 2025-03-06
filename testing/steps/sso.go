package steps

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/AlaudaDevops/bdd/logger"
	"github.com/cucumber/godog"
	"github.com/playwright-community/playwright-go"
	"go.uber.org/zap"
)

type ssoParams struct {
	ACPBaseURL  string        `yaml:"acpURL"`
	ACPUser     string        `yaml:"acpUser"`
	ACPPassword string        `yaml:"acpPassword"`
	SonarURL    string        `yaml:"sonarURL"`
	Timeout     time.Duration `yaml:"timeout"`
	Headless    bool          `yaml:"headless"`
}

func checkSSo(ctx context.Context, params *godog.DocString) (ctx2 context.Context, err error) {
	ssoParams := ssoParams{}
	if err := yaml.Unmarshal([]byte(params.Content), &ssoParams); err != nil {
		return ctx, err
	}

	return process(ctx, ssoParams)
}

func process(ctx context.Context, ssoParams ssoParams) (ctx2 context.Context, err error) {
	log := logger.LoggerFromContext(ctx)

	if ssoParams.Timeout == 0 {
		ssoParams.Timeout = 10 * time.Minute
	}

	// 安装 playwright
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	}); err != nil {
		log.Error("安装 playwright 失败", zap.Error(err))
		return ctx, err
	}

	// 初始化 playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Error("无法启动 playwright", zap.Error(err))
		return ctx, err
	}
	defer pw.Stop()

	// 启动浏览器
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(ssoParams.Headless),
		Args:     []string{"--ignore-certificate-errors"},
	})
	if err != nil {
		log.Error("无法启动浏览器", zap.Error(err))
		return ctx, err
	}
	defer browser.Close()

	// 创建新的上下文
	browserCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true),
	})
	if err != nil {
		log.Error("创建浏览器上下文失败", zap.Error(err))
		return ctx, err
	}
	defer browserCtx.Close()

	// 创建新的页面
	page, err := browserCtx.NewPage()
	if err != nil {
		log.Error("创建新页面失败: %v", zap.Error(err))
		return ctx, err
	}

	screenshotPath := "output/images/sonarqube-sso-screenshot.png"
	defer func() {
		data, screenshotErr := page.Screenshot(playwright.PageScreenshotOptions{
			Path: playwright.String(screenshotPath),
		})
		if screenshotErr != nil {
			log.Error("截图失败", zap.Error(screenshotErr))
		} else {
			ctx2 = godog.Attach(ctx2, godog.Attachment{
				Body:      data,
				FileName:  "sonarqube-sso-screenshot.png",
				MediaType: "image/png",
			})
			log.Info("保存截图成功", zap.String("path", screenshotPath))
		}
	}()

	// 执行登录流程
	if err := loginACP(ctx, page, ssoParams); err != nil {
		log.Error("ACP 登录失败", zap.Error(err))
		return ctx, err
	}

	if err := loginSonarqube(ctx, page, ssoParams); err != nil {
		log.Error("Sonarqube 登录失败: %v", zap.Error(err))
		return ctx, err
	}

	// 成功截图
	return ctx, nil
}

func loginACP(ctx context.Context, page playwright.Page, params ssoParams) error {
	log := logger.LoggerFromContext(ctx)

	log.Info("正在登录 acp...")

	if _, err := page.Goto(params.ACPBaseURL); err != nil {
		return fmt.Errorf("导航到登录页面失败: %v", err)
	}

	// 等待页面加载完成
	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	}); err != nil {
		return err
	}

	log.Info("等待登录表单出现...")
	if _, err := page.WaitForSelector(".login-form", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(30000),
	}); err != nil {
		return fmt.Errorf("等待登录表单: %v", err)
	}

	// 填写登录表单
	if err := page.Locator("input[name=\"username\"]").Fill(params.ACPUser); err != nil {
		return fmt.Errorf("填写用户名失败: %v", err)
	}

	if err := page.Locator("input[name=\"password\"]").Fill(params.ACPPassword); err != nil {
		return fmt.Errorf("填写密码失败: %v", err)
	}

	// 点击登录按钮
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{}).Click(); err != nil {
		return fmt.Errorf("点击登录按钮失败: %v", err)
	}

	// 等待 Devops 文本出现
	if err := page.Locator(fmt.Sprintf("//acl-page-header//div[text()='%v']", params.ACPUser)).WaitFor(); err != nil {
		return fmt.Errorf("等待 登录用户 文本出现失败: %v", err)
	}

	log.Info("acp 登录成功...")
	return nil
}

func loginSonarqube(ctx context.Context, page playwright.Page, params ssoParams) error {
	log := logger.LoggerFromContext(ctx)

	log.Info("正在登录 Sonarqube...")

	if _, err := page.Goto(params.SonarURL); err != nil {
		return fmt.Errorf("导航到 Sonarqube 登录页面失败: %v", err)
	}

	found := false
	timeout := time.After(params.Timeout)

	for !found {
		select {
		case <-timeout:
			return fmt.Errorf("等待 Log in with OpenID Connect 按钮超时")
		default:
			// 等待页面加载完成
			if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
				State: playwright.LoadStateNetworkidle,
			}); err != nil {
				return err
			}
			// 等待登录页面元素加载
			log.Info("等待 Log in with OpenID Connect 按钮出现...")
			if _, err := page.WaitForSelector(".identity-provider-link", playwright.PageWaitForSelectorOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(30000),
			}); err == nil {
				found = true
				break
			}

			if _, err := page.Reload(); err != nil {
				return err
			}
		}
	}

	log.Info("点击 Log in with OpenID Connect 按钮...")
	if err := page.Click(".identity-provider-link"); err != nil {
		return fmt.Errorf("点击 Log in with OpenID Connect 按钮失败: %v", err)
	}

	// Wait for page load to complete
	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	}); err != nil {
		return err
	}

	// Wait for URL to change to login success state
	// When user has no email, redirects to `user_settings/profile` page
	// When user has email, redirects to `dashboard/projects` page
	if err := page.WaitForURL(regexp.MustCompile(`.*(projects/create|projects)`), playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		return fmt.Errorf("等待 OIDC 表单失败: %v", err)
	}

	log.Info("测试成功！")
	return nil
}
