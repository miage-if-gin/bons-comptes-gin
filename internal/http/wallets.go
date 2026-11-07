package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/miage-if-gin/bons-comptes-gin/internal/domain"
	"github.com/miage-if-gin/bons-comptes-gin/internal/service"
)

// DashboardRow est une ligne du tableau de bord « Mes porte-monnaies ».
type DashboardRow struct {
	ID                int64
	Name              string
	WalletStatus      string
	MemberStatus      string
	MemberCount       int
	TotalExpenseCents int64
}

// Total formate le montant pour le template. Go n'autorise pas de méthode
// sur un type déclaré à l'intérieur d'une fonction, d'où la sortie au
// niveau du package.
func (r DashboardRow) Total() string { return service.FormatMontant(r.TotalExpenseCents) }

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

	private.GET("/wallets/:id", func(c *gin.Context) {
		walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "ID invalide")
			return
		}

		currentAccount := CompteCourant(c)

		var wallet domain.Wallet
		if err := d.DB.First(&wallet, walletID).Error; err != nil {
			c.String(http.StatusNotFound, "Porte-monnaie introuvable")
			return
		}

		var member domain.WalletMember
		if err := d.DB.Where("wallet_id = ? AND account_id = ?", walletID, currentAccount.ID).First(&member).Error; err != nil {
			c.String(http.StatusForbidden, "Vous n'avez pas accès à ce porte-monnaie")
			return
		}
		// Refuser une invitation retire l'accès : sans ce test, seule
		// l'existence de la ligne était vérifiée, pas son statut.
		if member.Status == "declined" {
			c.String(http.StatusForbidden, "Vous avez refusé l'invitation à ce porte-monnaie")
			return
		}

		d.Renderer.Render(c, http.StatusOK, "wallets_show.html", gin.H{
			"Title":  wallet.Name,
			"Wallet": wallet,
			"Member": member,
		})
	})

	private.POST("/wallets/:id/invites", func(c *gin.Context) {
		walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "ID de porte-monnaie invalide")
			return
		}

		email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
		if email == "" {
			c.String(http.StatusBadRequest, "L'email est requis")
			return
		}

		currentAccount := CompteCourant(c)

		var wallet domain.Wallet
		if err := d.DB.First(&wallet, walletID).Error; err != nil {
			c.String(http.StatusNotFound, "Porte-monnaie introuvable")
			return
		}
		if wallet.Status == "closed" {
			c.String(http.StatusForbidden, "Impossible d'inviter : ce porte-monnaie est clos")
			return
		}

		var currentMember domain.WalletMember
		if err := d.DB.Where("wallet_id = ? AND account_id = ?", walletID, currentAccount.ID).First(&currentMember).Error; err != nil {
			c.String(http.StatusForbidden, "Vous ne faites pas partie de ce porte-monnaie")
			return
		}
		if currentMember.Status != "accepted" {
			c.String(http.StatusForbidden, "Seuls les membres acceptés peuvent envoyer des invitations")
			return
		}

		var invitee domain.Account
		if err := d.DB.Where("email = ?", email).First(&invitee).Error; err != nil {
			c.String(http.StatusNotFound, "Aucun compte existant ne correspond à cet email")
			return
		}

		now := time.Now()
		newMember := domain.WalletMember{
			WalletID:  walletID,
			AccountID: invitee.ID,
			Status:    "pending",
			InvitedBy: &currentAccount.ID,
			InvitedAt: now,
		}

		if err := d.DB.Create(&newMember).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.String(http.StatusConflict, "Cet utilisateur est déjà membre ou a déjà été invité")
				return
			}
			c.String(http.StatusInternalServerError, "Erreur technique lors de l'invitation")
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/wallets/%d", walletID))
	})

	private.GET("/wallets", func(c *gin.Context) {
		currentAccount := CompteCourant(c)

		var rows []DashboardRow

		err := d.DB.Table("wallets w").
			Select(`
        				w.id,
        				w.name,
        				w.status as wallet_status,
        				wm.status as member_status,
        				(SELECT COUNT(account_id) FROM wallet_members WHERE wallet_id = w.id AND status = 'accepted') as member_count,
        				(SELECT COALESCE(SUM(amount_cents), 0)::BIGINT FROM expenses WHERE wallet_id = w.id) as total_expense_cents
        			`).
			Joins("JOIN wallet_members wm ON wm.wallet_id = w.id").
			Where("wm.account_id = ?", currentAccount.ID).
			Where("wm.status IN (?, ?)", "pending", "accepted").
			Scan(&rows).Error

		if err != nil {
			c.String(http.StatusInternalServerError, "Erreur lors de la récupération des données")
			return
		}

		var pending, open, closed []DashboardRow
		for _, r := range rows {
			if r.MemberStatus == "pending" {
				pending = append(pending, r)
			} else if r.WalletStatus == "open" {
				open = append(open, r)
			} else if r.WalletStatus == "closed" {
				closed = append(closed, r)
			}
		}

		d.Renderer.Render(c, http.StatusOK, "wallets_index.html", gin.H{
			"Title":   "Mes porte-monnaies",
			"Pending": pending,
			"Open":    open,
			"Closed":  closed,
		})
	})

	private.POST("/wallets/:id/respond", func(c *gin.Context) {
		walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "ID invalide")
			return
		}

		action := c.PostForm("action")
		if action != "accept" && action != "decline" {
			c.String(http.StatusBadRequest, "Action non reconnue")
			return
		}

		currentAccount := CompteCourant(c)

		var member domain.WalletMember
		if err := d.DB.Where("wallet_id = ? AND account_id = ?", walletID, currentAccount.ID).First(&member).Error; err != nil {
			c.String(http.StatusForbidden, "Invitation introuvable ou vous n'y avez pas accès")
			return
		}

		if member.Status != "pending" {
			c.String(http.StatusBadRequest, "Cette invitation a déjà été traitée")
			return
		}

		newStatus := "accepted"
		if action == "decline" {
			newStatus = "declined"
		}

		now := time.Now()
		member.Status = newStatus
		member.RespondedAt = &now

		if err := d.DB.Save(&member).Error; err != nil {
			c.String(http.StatusInternalServerError, "Erreur lors de l'enregistrement de la réponse")
			return
		}

		c.Redirect(http.StatusSeeOther, "/wallets")
	})
}
