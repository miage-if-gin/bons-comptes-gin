package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
)

func registerWallets(private *gin.RouterGroup, d Deps) {

    private.GET("/wallets/new", func(c *gin.Context) {
        d.Renderer.Render(c, http.StatusOK, "wallets_new.html", gin.H{
            "Title": "Nouveau Porte-monnaie",
        })
    })

	private.POST("/wallets", func(c *gin.Context) {
		walletName := c.PostForm("name")
		if walletName == "" {
			c.String(http.StatusBadRequest, "Le nom du porte-monnaie est requis")
			return
		}

        currentAccount := CompteCourant(c)
		if currentAccount == nil {
			c.String(http.StatusUnauthorized, "Non autorisé")
			return
		}

		var newWalletID int64

		err := d.DB.Transaction(func(tx *gorm.DB) error {
			wallet := domain.Wallet{
				Name:    walletName,
				OwnerID: currentAccount.ID,
				Status:  "open",
			}
			if err := tx.Create(&wallet).Error; err != nil {
				return err
			}

			now := time.Now()
			member := domain.WalletMember{
				WalletID:    wallet.ID,
				AccountID:   currentAccount.ID,
				Status:      "accepted",
				RespondedAt: &now,
			}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}

			newWalletID = wallet.ID
			return nil
		})

		if err != nil {
			c.String(http.StatusInternalServerError, "Erreur lors de la création du porte-monnaie")
			return
		}

		c.Redirect(http.StatusFound, fmt.Sprintf("/wallets/%d", newWalletID))
	})
}