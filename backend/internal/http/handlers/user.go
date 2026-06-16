package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bitapi/backend/internal/http/middleware"
	bithttp "bitapi/backend/internal/http/respond"
	"bitapi/backend/internal/models"
	keysvc "bitapi/backend/internal/services/keys"
	paymentsvc "bitapi/backend/internal/services/payments"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	keys     *keysvc.Service
	payments *paymentsvc.Service
	db       *gorm.DB
}

func NewUserHandler(keys *keysvc.Service, payments *paymentsvc.Service, db *gorm.DB) *UserHandler {
	return &UserHandler{keys: keys, payments: payments, db: db}
}

const maxAvatarBytes = 2 << 20

func (h *UserHandler) UploadAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarBytes+1024)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "请选择头像图片")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxAvatarBytes {
		bithttp.Fail(c, http.StatusBadRequest, "头像图片不能超过 2MB")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "头像图片读取失败")
		return
	}
	head := make([]byte, 512)
	n, _ := file.Read(head)
	_ = file.Close()
	contentType := http.DetectContentType(head[:n])
	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}
	ext, ok := extensions[contentType]
	if !ok {
		bithttp.Fail(c, http.StatusBadRequest, "头像仅支持 JPG、PNG、GIF、WEBP 图片")
		return
	}

	randomPart, err := randomHex(8)
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	userID := c.GetUint(middleware.ContextUserID)
	fileName := fmt.Sprintf("%d-%d-%s%s", userID, time.Now().UnixNano(), randomPart, ext)
	uploadDir := filepath.Join("data", "uploads", "avatars")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	targetPath := filepath.Join(uploadDir, fileName)
	if err := c.SaveUploadedFile(fileHeader, targetPath); err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"avatar_url": "/uploads/avatars/" + fileName})
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	displayName := strings.TrimSpace(req.DisplayName)
	avatarURL := strings.TrimSpace(req.AvatarURL)
	if displayName == "" {
		bithttp.Fail(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if len([]rune(displayName)) > 60 {
		bithttp.Fail(c, http.StatusBadRequest, "用户名不能超过 60 个字符")
		return
	}
	if len(avatarURL) > 500 {
		bithttp.Fail(c, http.StatusBadRequest, "头像地址不能超过 500 个字符")
		return
	}

	userID := c.GetUint(middleware.ContextUserID)
	updates := map[string]any{
		"display_name": displayName,
		"avatar_url":   avatarURL,
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		bithttp.Fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	bithttp.OK(c, user)
}

func (h *UserHandler) ListKeys(c *gin.Context) {
	keys, err := h.keys.List(c.GetUint(middleware.ContextUserID))
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, keys)
}

func (h *UserHandler) CreateKey(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		GroupID          *uint  `json:"group_id"`
		QuotaLimitMicros int64  `json:"quota_limit_micros"`
		ExpiresAt        string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			bithttp.Fail(c, http.StatusBadRequest, "过期时间必须使用 RFC3339 格式")
			return
		}
		expiresAt = &parsed
	}
	created, err := h.keys.Create(c.GetUint(middleware.ContextUserID), req.Name, req.GroupID, req.QuotaLimitMicros, expiresAt)
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.Created(c, created)
}

func (h *UserHandler) DeleteKey(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, "编号无效")
		return
	}
	if err := h.keys.Delete(c.GetUint(middleware.ContextUserID), uint(id64)); err != nil {
		bithttp.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	bithttp.OK(c, gin.H{"deleted": true})
}

func (h *UserHandler) Usage(c *gin.Context) {
	userID := c.GetUint(middleware.ContextUserID)
	var rows []map[string]any
	err := h.db.Table("usage_logs").Where("user_id = ?", userID).Order("id desc").Limit(100).Find(&rows).Error
	if err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *UserHandler) Orders(c *gin.Context) {
	var rows []models.PaymentOrder
	if err := h.db.Where("user_id = ?", c.GetUint(middleware.ContextUserID)).Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		bithttp.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	bithttp.OK(c, rows)
}

func (h *UserHandler) CreateOrder(c *gin.Context) {
	if !h.settingEnabled("module.payment.order.enabled", true) {
		bithttp.Fail(c, http.StatusForbidden, "充值订单功能已关闭")
		return
	}
	var req struct {
		AmountMicros int64  `json:"amount_micros" binding:"required"`
		Provider     string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AmountMicros <= 0 {
		bithttp.Fail(c, http.StatusBadRequest, "金额必须大于 0")
		return
	}
	intent, err := h.payments.CreatePaymentIntent(c.GetUint(middleware.ContextUserID), req.AmountMicros, req.Provider)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.Created(c, intent)
}

func (h *UserHandler) Redeem(c *gin.Context) {
	if !h.settingEnabled("module.payment.redeem.enabled", true) {
		bithttp.Fail(c, http.StatusForbidden, "兑换码功能已关闭")
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	usage, err := h.payments.Redeem(c.GetUint(middleware.ContextUserID), req.Code)
	if err != nil {
		bithttp.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	bithttp.Created(c, usage)
}

func (h *UserHandler) settingEnabled(key string, fallback bool) bool {
	var setting struct {
		Value string
	}
	if err := h.db.Table("settings").Select("value").Where("key = ?", key).Take(&setting).Error; err != nil {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(setting.Value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return fallback
	}
}
