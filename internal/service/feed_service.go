package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	// 按发布日期分组存储文章
	articlesByDate := make(map[string][]*gofeed.Item)

	for _, item := range parsedFeed.Items {
		// 获取文章发布日期
		publishDate := time.Now().Format("2006-01-02")
		if item.PublishedParsed != nil {
			publishDate = item.PublishedParsed.Format("2006-01-02")
		}
		articlesByDate[publishDate] = append(articlesByDate[publishDate], item)
	}

	// 处理每个日期的文章
	totalNew := 0
	for date, items := range articlesByDate {
		// 确保日期目录存在
		if err := s.cfgMgr.EnsureRSSDateDir(date); err != nil {
			log.Warn().Err(err).Str("date", date).Msg("Failed to create date directory")
			continue
		}

		// 加载或创建该日期的overview
		overview, err := s.LoadOverview(date)
		if err != nil {
			log.Warn().Err(err).Str("date", date).Msg("Failed to load overview")
			continue
		}
		if overview == nil {
			overview = &model.DailyOverview{
				Date:       date,
				UpdatedAt:  time.Now(),
				TotalCount: 0,
				Articles:   []model.ArticleSummary{},
			}
		}

		// 创建已存在文章ID的map，用于去重
		existingIDs := make(map[string]bool)
		for _, article := range overview.Articles {
			existingIDs[article.ID] = true
		}

		// 处理该日期的文章
		newCount := 0
		for _, item := range items {
			articleID := generateArticleID(item.Link)

			// 跳过已存在的文章
			if existingIDs[articleID] {
				continue
			}

			// 解析发布时间
			published := time.Now()
			if item.PublishedParsed != nil {
				published = *item.PublishedParsed
			}

			// 获取完整内容（用于摘要）
			rawContent := item.Description
			if item.Content != "" {
				rawContent = item.Content
			}

			// 将 HTML 转换为可读的纯文本
			readableText := convertToReadableText(item, feed.Name)

			// 保存可读文本到文件
			contentPath := s.cfgMgr.RSSContentPath(date, articleID)
			if err := os.WriteFile(contentPath, []byte(readableText), 0644); err != nil {
				log.Warn().Err(err).Str("article", articleID).Msg("Failed to save content file")
				continue
			}

			// 生成纯文本摘要
			plainSummary := html2text.HTML2Text(rawContent)
			if plainSummary == "" {
				plainSummary = rawContent
			}

			// 创建摘要
			summary := model.ArticleSummary{
				ID:          articleID,
				FeedID:      feed.ID,
				FeedName:    feed.Name,
				Title:       item.Title,
				Link:        item.Link,
				Summary:     truncateContent(plainSummary, 200),
				ContentFile: contentPath,
				Published:   published,
				Fetched:     time.Now(),
			}

			overview.Articles = append(overview.Articles, summary)
			existingIDs[articleID] = true
			newCount++
		}

		// 更新overview
		if newCount > 0 {
			overview.UpdatedAt = time.Now()
			overview.TotalCount = len(overview.Articles)
			if err := s.SaveOverview(date, overview); err != nil {
				log.Warn().Err(err).Str("date", date).Msg("Failed to save overview")
			}
			log.Info().Str("feed", feed.Name).Str("date", date).Int("new_articles", newCount).Msg("Articles saved")
			totalNew += newCount
		}
	}

	if totalNew > 0 {
		log.Info().Str("feed", feed.Name).Int("total_new", totalNew).Msg("Feed fetched successfully")
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
	today := time.Now().Format("2006-01-02")
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
