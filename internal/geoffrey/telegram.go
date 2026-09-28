package geoffrey

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (a *App) RunTelegram() error {
	if strings.TrimSpace(a.cfg.TelegramBotToken) == "" {
		return fmt.Errorf("telegram bot token is not configured")
	}
	bot, err := tgbotapi.NewBotAPI(a.cfg.TelegramBotToken)
	if err != nil {
		return err
	}
	bot.Debug = false
	log.Printf("geoffrey: telegram bot authorized as %s", bot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		text := strings.TrimSpace(update.Message.Text)
		if text == "" {
			continue
		}
		reply := a.handleTelegramText(text)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		_, _ = bot.Send(msg)
	}
	return nil
}

func (a *App) handleTelegramText(text string) string {
	if actionReply := a.handleTelegramAction(text); actionReply != "" {
		return actionReply
	}
	if naturalReply := a.handleNaturalCollectionIntent(text); naturalReply != "" {
		return naturalReply
	}
	low := strings.ToLower(strings.TrimSpace(text))
	switch {
	case low == "/start":
		return "Soy Geoffrey. Puedo ayudarte con colecciones de Plex. Prueba: /libraries, /recipes, /collections Películas, /search Películas|King Kong"
	case low == "/libraries":
		libs, err := a.Libraries()
		if err != nil {
			return "Error cargando bibliotecas: " + err.Error()
		}
		if len(libs) == 0 {
			return "No veo bibliotecas Plex."
		}
		var lines []string
		for _, lib := range libs {
			lines = append(lines, fmt.Sprintf("- %s (%s, key=%s)", lib.Title, lib.Type, lib.Key))
		}
		return strings.Join(lines, "\n")
	case low == "/recipes":
		recipes := a.Recipes()
		if len(recipes) == 0 {
			return "No hay recetas cargadas."
		}
		var lines []string
		for _, recipe := range recipes {
			lines = append(lines, fmt.Sprintf("- %s (%s)", recipe.Name, recipe.ID))
		}
		return strings.Join(lines, "\n")
	case strings.HasPrefix(low, "/collections"):
		arg := strings.TrimSpace(strings.TrimPrefix(text, "/collections"))
		lib := a.cfg.PlexDefaultLibrary
		if arg != "" {
			lib = arg
		}
		resolved, err := a.ResolveLibrarySection(lib)
		if err != nil {
			return "No encuentro la biblioteca: " + err.Error()
		}
		items, err := a.Collections(resolved.Key)
		if err != nil {
			return "Error listando colecciones: " + err.Error()
		}
		if len(items) == 0 {
			return "No veo colecciones en esa biblioteca."
		}
		var lines []string
		for _, item := range items {
			lines = append(lines, fmt.Sprintf("- %s (%d items)", item.Title, item.ChildCount))
		}
		return strings.Join(lines, "\n")
	default:
		return "Aún estoy verde 😅\nUsa por ahora: /libraries, /recipes, /collections [biblioteca], /search <biblioteca>|<texto>, /create_collection <biblioteca>|<nombre>|<titulo1, titulo2>"
	}
}
