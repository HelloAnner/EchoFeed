package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// FeedService RSS订阅服务
type FeedService struct {
	cfgMgr *config.Manager
	parser *gofeed.Parser
	mu     sync.RWMutex
}

// NewFeedService 创建RSS订阅服务
func NewFeedService(cfgMgr *config.Manager) *FeedService {
	return &FeedService{
		cfgMgr: cfgMgr,
		parser: gofeed.NewParser(),
	}
}

// List 获取所有订阅
func (s *FeedService) List() ([]model.Feed, error) {
	cfg, err := s.cfgMgr.LoadFeeds()
	if err != nil {
		return nil, err
	}
	return cfg.Feeds, nil
}

// Get 获取单个订阅
func (s *FeedService) Get(id string) (*model.Feed, error) {
	cfg, err := s.cfgMgr.LoadFeeds()
	if err != nil {
		return nil, err
	}
	for _, f := range cfg.Feeds {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, nil
}

// Create 创建订阅
func (s *FeedService) Create(feed *model.Feed) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadFeeds()
	if err != nil {
		return err
	}

	if feed.ID == "" {
		feed.ID = uuid.New().String()[:8]
	}
	cfg.Feeds = append(cfg.Feeds, *feed)
	return s.cfgMgr.SaveFeeds(cfg)
}

// Update 更新订阅
func (s *FeedService) Update(feed *model.Feed) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadFeeds()
	if err != nil {
		return err
	}

	for i, f := range cfg.Feeds {
		if f.ID == feed.ID {
			cfg.Feeds[i] = *feed
			return s.cfgMgr.SaveFeeds(cfg)
		}
	}
	return nil
}

// Delete 删除订阅
func (s *FeedService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadFeeds()
	if err != nil {
		return err
	}

	for i, f := range cfg.Feeds {
		if f.ID == id {
			cfg.Feeds = append(cfg.Feeds[:i], cfg.Feeds[i+1:]...)
			return s.cfgMgr.SaveFeeds(cfg)
		}
	}
	return nil
}

// GetSettings 获取RSS设置
func (s *FeedService) GetSettings() (*model.FeedSettings, error) {
	cfg, err := s.cfgMgr.LoadFeeds()
	if err != nil {
		return nil, err
	}
	return &cfg.Settings, nil
}

// TestFeedURL 测试RSS订阅URL连通性与可解析性（用于保存前校验）
func (s *FeedService) TestFeedURL(url string) (string, int, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return "", 0, err
	}

	timeoutSec := settings.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build_request: %w", err)
	}
	if settings.UserAgent != "" {
		req.Header.Set("User-Agent", settings.UserAgent)
	}

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", 0, fmt.Errorf("http_status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", 0, fmt.Errorf("read: %w", err)
	}

	parser := gofeed.NewParser()
	parsed, err := parser.ParseString(string(body))
	if err != nil {
		return "", 0, fmt.Errorf("parse: %w", err)
	}

	title := strings.TrimSpace(parsed.Title)
	return title, len(parsed.Items), nil
}

// generateArticleID 生成文章ID (基于链接的MD5前8位)
func generateArticleID(link string) string {
	hash := md5.Sum([]byte(link))
	return hex.EncodeToString(hash[:])[:8]
}

// truncateContent 截取内容摘要
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// convertToReadableText 将 HTML 内容转换为可读的纯文本格式（类似 newsboat）
func convertToReadableText(item *gofeed.Item, feedName string) string {
	var sb strings.Builder

	// 文章头部信息
	sb.WriteString("=" + strings.Repeat("=", 70) + "\n")
	sb.WriteString(fmt.Sprintf("Title: %s\n", item.Title))
	sb.WriteString(fmt.Sprintf("Feed: %s\n", feedName))
	if item.Link != "" {
		sb.WriteString(fmt.Sprintf("Link: %s\n", item.Link))
	}
	if item.PublishedParsed != nil {
		sb.WriteString(fmt.Sprintf("Date: %s\n", item.PublishedParsed.Format("2006-01-02 15:04:05")))
	}
	if len(item.Authors) > 0 && item.Authors[0].Name != "" {
		sb.WriteString(fmt.Sprintf("Author: %s\n", item.Authors[0].Name))
	}
	sb.WriteString("=" + strings.Repeat("=", 70) + "\n\n")

	// 获取内容
	content := item.Description
	if item.Content != "" {
		content = item.Content
	}

	// 将 HTML 转换为纯文本
	if content != "" {
		text := html2text.HTML2Text(content)
		sb.WriteString(text)
	}

	sb.WriteString("\n\n")
	sb.WriteString("-" + strings.Repeat("-", 70) + "\n")

	return sb.String()
}

// Fetch 拉取单个订阅并保存到日期目录（按文章发布日期归档）
func (s *FeedService) Fetch(feed *model.Feed) error {
	log.Info().Str("feed", feed.Name).Str("url", feed.URL).Msg("Fetching feed")

	parsedFeed, err := s.parser.ParseURL(feed.URL)
	if err != nil {
		log.Error().Err(err).Str("feed", feed.Name).Msg("Failed to fetch feed")
		return err
	}

	today := time.Now().In(time.Local).Format("2006-01-02")
	yesterday := time.Now().In(time.Local).AddDate(0, 0, -1).Format("2006-01-02")

	if err := s.cfgMgr.EnsureRSSDateDir(today); err != nil {
		return err
	}

	overview, err := s.LoadOverview(today)
	if err != nil {
		return err
	}
	if overview == nil {
		overview = &model.DailyOverview{
			Date:       today,
			UpdatedAt:  time.Now().In(time.Local),
			TotalCount: 0,
			Articles:   []model.ArticleSummary{},
		}
	}

	existingToday := make(map[string]bool)
	for _, a := range overview.Articles {
		existingToday[a.ID] = true
	}

	var existingYesterday map[string]bool
	loadYesterday := func() {
		if existingYesterday != nil {
			return
		}
		existingYesterday = make(map[string]bool)
		prev, err := s.LoadOverview(yesterday)
		if err != nil || prev == nil {
			return
		}
		for _, a := range prev.Articles {
			existingYesterday[a.ID] = true
		}
	}

	newCount := 0
	for _, item := range parsedFeed.Items {
		articleID := generateArticleID(item.Link)
		if existingToday[articleID] {
			continue
		}

		published := time.Now().In(time.Local)
		if item.PublishedParsed != nil {
			published = item.PublishedParsed.In(time.Local)
		}
		publishDate := published.Format("2006-01-02")

		// 只关注今天；兜底：昨天的文章如果昨天目录没收录，则归档到今天(但发布时间保留为昨天/今天)
		if publishDate != today {
			if publishDate != yesterday {
				continue
			}
			loadYesterday()
			if existingYesterday[articleID] {
				continue
			}
		}

		rawContent := item.Description
		if item.Content != "" {
			rawContent = item.Content
		}

		readableText := convertToReadableText(item, feed.Name)
		contentPath := s.cfgMgr.RSSContentPath(today, articleID)
		if err := os.WriteFile(contentPath, []byte(readableText), 0644); err != nil {
			log.Warn().Err(err).Str("article", articleID).Msg("Failed to save content file")
			continue
		}

		plainSummary := html2text.HTML2Text(rawContent)
		if plainSummary == "" {
			plainSummary = rawContent
		}

		overview.Articles = append(overview.Articles, model.ArticleSummary{
			ID:          articleID,
			FeedID:      feed.ID,
			FeedName:    feed.Name,
			Title:       item.Title,
			Link:        item.Link,
			Summary:     truncateContent(plainSummary, 200),
			ContentFile: contentPath,
			Published:   published,
			Fetched:     time.Now().In(time.Local),
		})
		existingToday[articleID] = true
		newCount++
	}

	if newCount > 0 {
		overview.UpdatedAt = time.Now().In(time.Local)
		overview.TotalCount = len(overview.Articles)
		if err := s.SaveOverview(today, overview); err != nil {
			return err
		}
		log.Info().Str("feed", feed.Name).Str("date", today).Int("new_articles", newCount).Msg("Articles saved")
	} else {
		log.Info().Str("feed", feed.Name).Msg("No new articles")
	}
	return nil
}

// FetchAll 拉取所有启用的订阅
func (s *FeedService) FetchAll() error {
	feeds, err := s.List()
	if err != nil {
		return err
	}

	settings, err := s.GetSettings()
	if err != nil {
		return err
	}

	// 使用信号量控制并发
	sem := make(chan struct{}, settings.MaxConcurrent)
	var wg sync.WaitGroup

	for _, feed := range feeds {
		if !feed.Enabled {
			continue
		}

		wg.Add(1)
		go func(f model.Feed) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := s.Fetch(&f); err != nil {
				log.Error().Err(err).Str("feed", f.Name).Msg("Failed to fetch feed")
			}
		}(feed)
	}

	wg.Wait()
	return nil
}

// LoadOverview 加载指定日期的overview
func (s *FeedService) LoadOverview(date string) (*model.DailyOverview, error) {
	path := s.cfgMgr.RSSOverviewPath(date)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var overview model.DailyOverview
	if err := json.Unmarshal(data, &overview); err != nil {
		return nil, err
	}
	return &overview, nil
}

// SaveOverview 保存指定日期的overview
func (s *FeedService) SaveOverview(date string, overview *model.DailyOverview) error {
	path := s.cfgMgr.RSSOverviewPath(date)
	data, err := json.MarshalIndent(overview, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadArticleContent 加载文章完整内容
func (s *FeedService) LoadArticleContent(date, articleID string) (string, error) {
	path := s.cfgMgr.RSSContentPath(date, articleID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTodayOverview 获取今日overview
func (s *FeedService) GetTodayOverview() (*model.DailyOverview, error) {
	today := time.Now().In(time.Local).Format("2006-01-02")
	return s.LoadOverview(today)
}

// GetArticlesByIDs 根据ID列表获取文章摘要
func (s *FeedService) GetArticlesByIDs(date string, ids []string) ([]model.ArticleSummary, error) {
	overview, err := s.LoadOverview(date)
	if err != nil || overview == nil {
		return nil, err
	}

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	var result []model.ArticleSummary
	for _, article := range overview.Articles {
		if idSet[article.ID] {
			result = append(result, article)
		}
	}
	return result, nil
}

// ---- 以下为旧版兼容方法，保留供迁移期间使用 ----

// SaveCache 保存RSS缓存 (旧版兼容)
func (s *FeedService) SaveCache(feedID string, cache *model.FeedCache) error {
	path := s.cfgMgr.RSSCachePath(feedID)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadCache 加载RSS缓存 (旧版兼容)
func (s *FeedService) LoadCache(feedID string) (*model.FeedCache, error) {
	path := s.cfgMgr.RSSCachePath(feedID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cache model.FeedCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// LoadAllCaches 加载所有RSS缓存 (旧版兼容)
func (s *FeedService) LoadAllCaches() ([]model.FeedCache, error) {
	feeds, err := s.List()
	if err != nil {
		return nil, err
	}

	var caches []model.FeedCache
	for _, feed := range feeds {
		if !feed.Enabled {
			continue
		}
		cache, err := s.LoadCache(feed.ID)
		if err != nil {
			log.Warn().Err(err).Str("feed", feed.Name).Msg("Failed to load cache")
			continue
		}
		if cache != nil {
			caches = append(caches, *cache)
		}
	}
	return caches, nil
}
