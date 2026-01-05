package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

type QRCodeController struct {
	baseURL string
}

func NewQRCodeController(baseURL string) *QRCodeController {
	return &QRCodeController{
		baseURL: baseURL,
	}
}

// GenerateQRCode handles GET /api/v1/qrcode/:shortCode - generates QR code for a short URL
func (qc *QRCodeController) GenerateQRCode(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Short code is required",
		})
		return
	}

	// Construct the full short URL
	shortURL := qc.baseURL + "/" + shortCode

	// Generate QR code (256x256 pixels, medium error recovery)
	qrCode, err := qrcode.New(shortURL, qrcode.Medium)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate QR code",
		})
		return
	}

	// Convert to PNG
	pngData, err := qrCode.PNG(256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate QR code image",
		})
		return
	}

	// Set headers and return image
	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=qrcode.png")
	c.Data(http.StatusOK, "image/png", pngData)
}

