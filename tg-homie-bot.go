package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-telegram-bot-api/telegram-bot-api"

	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
)

type hookConfig struct {
	updateID int
	chatID   int64
	userID   int

	hasText     bool
	hasPhoto    bool
	hasCallback bool

	inTable bool
	status  string
}

var (
	// bot
	bot      *tgbotapi.BotAPI
	botToken = "1043743898:AAFLHzukT06kDialBm0XcNJREFjEYf10ErM"
	baseURL  = "https://tg-homie-bot.herokuapp.com/"

	// DB
	dbURI = "postgres://tkkghsfsvqtsdd:0a879d2b89e3cd6b7fbd4f827cfa8eb5dfabd938aefe0e519c3efd1e10255bdd@ec2-18-232-143-90.compute-1.amazonaws.com:5432/d6372psrheqgci"
	db, _ = sql.Open("postgres", dbURI)
)

// Парсер Update
func parseUpdate(update tgbotapi.Update) hookConfig {
	if update.CallbackQuery != nil {
		hook := hookConfig{
			updateID:    update.UpdateID,
			hasCallback: true,
			hasPhoto:    false,
			hasText:     false,
			chatID:      update.CallbackQuery.Message.Chat.ID,
			userID:      update.CallbackQuery.From.ID,
		}
		return hook
	} else if update.Message != nil {
		if update.Message.Photo != nil {
			hook := hookConfig{
				updateID:    update.UpdateID,
				hasCallback: false,
				hasPhoto:    true,
				hasText:     false,
				chatID:      update.Message.Chat.ID,
				userID:      update.Message.From.ID,
			}
			return hook
		}
		hook := hookConfig{
			updateID:    update.UpdateID,
			hasCallback: false,
			hasPhoto:    false,
			hasText:     true,
			chatID:      update.Message.Chat.ID,
			userID:      update.Message.From.ID,
		}
		return hook

	}

	hook := hookConfig{
		updateID:    update.UpdateID,
		hasCallback: false,
		hasPhoto:    false,
		hasText:     false,
		chatID:      update.Message.Chat.ID,
		userID:      update.Message.From.ID,
	}
	return hook
}

// Меню  в зависимости от role
func menuReply(role string) interface{} {
	// Решатель
	if role == "solver" {
		return tgbotapi.ReplyKeyboardMarkup{
			Keyboard: [][]tgbotapi.KeyboardButton{
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Мой профиль"),
					tgbotapi.NewKeyboardButton("Мои ответы на заявки"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Поиск заявок"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Удалить профиль"),
				),
			},
			OneTimeKeyboard: true,
			ResizeKeyboard:  true,
		}
	}
	// Спрашивающий
	return tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Мой профиль"),
				tgbotapi.NewKeyboardButton("Мои заявки"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Подать заявку"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Удалить профиль"),
			),
		},
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}
}

// Логика регистрации пользователя
func logicReg(hook hookConfig, update tgbotapi.Update) {
	switch hook.status {
	case "reg1": // role
		switch hook.hasCallback {
		case true:
			if update.CallbackQuery.Data == "Решателем" {
				newStatus(db, hook.userID, "reg2")
				setText(db, "bot_user", hook.userID, "role", "solver")
			} else if update.CallbackQuery.Data == "Спрашивателем" {
				newStatus(db, hook.userID, "reg2")
				setText(db, "bot_user", hook.userID, "role", "asking")
			}
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Ok, твое Имя?"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Есть только два типа людей 🎸"))
		}
	case "reg2": // name
		switch hook.hasText {
		case true:
			if len(update.Message.Text) >= 3 {
				newStatus(db, hook.userID, "reg3")
				setText(db, "bot_user", hook.userID, "name", update.Message.Text)
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Твоя Фамилия?"))
			} else {
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Таких коротких Имен не бывает ☹️"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста, отправьте текст"))
		}
	case "reg3": // surname
		switch hook.hasText {
		case true:
			if len(update.Message.Text) >= 3 {
				if getText(db, "bot_user", hook.userID, "role") == "solver" {
					newStatus(db, hook.userID, "reg35")
					setText(db, "bot_user", hook.userID, "surname", update.Message.Text)
					bot.Send(tgbotapi.NewMessage(hook.chatID, "Где вы учитесь? или работаете"))
				} else {
					newStatus(db, hook.userID, "reg4")
					setText(db, "bot_user", hook.userID, "surname", update.Message.Text)
					bot.Send(tgbotapi.NewMessage(hook.chatID, "Отправь фоточку 📷"))
				}
			} else {
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Таких коротких Фамилий не бывает ☹️"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста, отправьте текст"))
		}
	case "reg35": // level
		switch hook.hasText {
		case true:
			if len(update.Message.Text) >= 3 {
				newStatus(db, hook.userID, "reg4")
				setText(db, "bot_user", hook.userID, "level", update.Message.Text)
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Отправь фоточку 📷"))
			} else {
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста чуть больше буковок"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста, отправьте текст"))
		}
	case "reg4": // img
		switch hook.hasPhoto {
		case true:
			newStatus(db, hook.userID, "menu")
			setText(db, "bot_user", hook.userID, "img", (*update.Message.Photo)[0].FileID)
			menu := tgbotapi.NewMessage(hook.chatID, "Ура!🎉 Теперь ты зарегистрирован в социальной сети интернет 🌐. Выбирай что делать дальше.")
			menu.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			bot.Send(menu)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Фоточку 📷 дружище"))
		}
	}
}

// Логика регистрации заявки
func logicAsk(hook hookConfig, update tgbotapi.Update) {
	id := getInt(db, "bot_user", hook.userID, "lastask")
	switch hook.status {
	case "ask1": // urg
		switch hook.hasCallback {
		case true:
			if update.CallbackQuery.Data == "Срочная" {
				newStatus(db, hook.userID, "ask2")
				setText(db, "asking", id, "urg", "quick")
			} else if update.CallbackQuery.Data == "Не срочная" {
				newStatus(db, hook.userID, "ask2")
				setText(db, "asking", id, "urg", "slow")
			}
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Уровень вашей заявки"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Есть только два типа заявок"))
		}
	case "ask2": // level
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "ask3")
			setText(db, "asking", id, "level", update.Message.Text)
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Немного опишите заявку"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пришлите текст"))
		}
	case "ask3": // info
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "ask4")
			setText(db, "asking", id, "info", update.Message.Text)
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Сколько вы готовы заплатить за решение?"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пришлите текст"))
		}
	case "ask4": // price date
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "menu")
			setText(db, "asking", id, "price", update.Message.Text)

			time := time.Unix(int64(update.Message.Date), 0).Add(3 * time.Hour)
			setText(db, "asking", id, "date", strconv.Itoa(time.Day())+" "+time.Month().String()+" "+strconv.Itoa(time.Hour())+":"+strconv.Itoa(time.Minute()))

			menu := tgbotapi.NewMessage(hook.chatID, "Заявка принята")
			menu.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			bot.Send(menu)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пришлите текст"))
		}
	}
}

// Логика поиска заявки
func logicSearch(hook hookConfig, update tgbotapi.Update) {
	flag := false
	id := getInt(db, "bot_user", hook.userID, "lastask")
	rows, err := db.Query("SELECT id,urg,date,level,info,price FROM asking WHERE id <> " + strconv.Itoa(id) + " AND idSolv = 0;")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", hook.userID, err.Error())
	} else {
		for rows.Next() {
			var (
				id    int
				urg   string
				date  string
				level string
				info  string
				price string
	
				urgPre string
			)
			rows.Scan(&id, &urg, &date, &level, &info, &price)
	
			setInt(db,"bot_user",hook.userID,"lastask",id)
			if urg == "quick" {
				urgPre = "Срочная"
			}
			if urg == "slow" {
				urgPre = "Не срочная"
			}
			htmlText := `<b>` + urgPre + `</b>
	` + date + `
	<b>Уровень</b> ` + level + `
	<b>Информация</b> ` + info + `
	<b>Цена</b> ` + price
	
			NextEnd := tgbotapi.NewMessage(hook.chatID, htmlText)
			NextEnd.ParseMode = tgbotapi.ModeHTML
			keyboard := tgbotapi.InlineKeyboardMarkup{}
			var row []tgbotapi.InlineKeyboardButton
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Взять", "Взять"))
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Закончить", "Закончить"))
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Следующая", "Следующая"))
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
			NextEnd.ReplyMarkup = keyboard
			bot.Send(NextEnd)
			flag=true
			return
		}
		if flag == false {
			msg := tgbotapi.NewMessage(hook.chatID, "У вас еще нет заявок")
			msg.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			bot.Send(msg)
			return
		}
	}
}

func logicTake(hook hookConfig, update tgbotapi.Update) {
	id := getInt(db, "bot_user", hook.userID, "lastask")
	setInt(db,"asking",id,"idSolv",hook.userID)
}

// Профиль пользователя
func userProfile(hook hookConfig, update tgbotapi.Update) {
	role := getText(db, "bot_user", hook.userID, "role")
	img := tgbotapi.NewPhotoShare(hook.chatID, getText(db, "bot_user", hook.userID, "img"))
	bot.Send(img)

	if role == "solver" {
		bot.Send(tgbotapi.NewMessage(hook.chatID, "Решатель "+getText(db, "bot_user", hook.userID, "name")+" "+getText(db, "bot_user", hook.userID, "surname")))
		msg := tgbotapi.NewMessage(hook.chatID, "Уровень знаний "+getText(db, "bot_user", hook.userID, "level"))
		msg.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
		bot.Send(msg)
	}
	if role == "asking" {
		msg := tgbotapi.NewMessage(hook.chatID, "Спрашиватель "+getText(db, "bot_user", hook.userID, "name")+" "+getText(db, "bot_user", hook.userID, "surname"))
		msg.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
		bot.Send(msg)
	}
}

// Заявки спрашивателя
func userAsk(hook hookConfig, update tgbotapi.Update) {
	flag := false
	rows, err := db.Query("SELECT urg,date,level,info,price FROM asking WHERE idUser = " + strconv.Itoa(hook.userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", hook.userID, err.Error())
	} else {
		for rows.Next() {
			var (
				urg   string
				date  string
				level string
				info  string
				price string

				urgPre string
			)
			rows.Scan(&urg, &date, &level, &info, &price)

			if urg == "quick" {
				urgPre = "Срочная"
			}
			if urg == "slow" {
				urgPre = "Не срочная"
			}

			htmlText := `<b>` + urgPre + `</b>
` + date + `
<b>Уровень</b> ` + level + `
<b>Информация</b> ` + info + `
<b>Цена</b> ` + price

			ask := tgbotapi.NewMessage(hook.chatID, htmlText)
			ask.ParseMode = tgbotapi.ModeHTML
			ask.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			flag = true
			bot.Send(ask)
		}
		if flag == false {
			msg := tgbotapi.NewMessage(hook.chatID, "У вас еще нет заявок")
			msg.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			bot.Send(msg)
			return
		}
	}
}

// Заявки спрашивателя
func userSolv(hook hookConfig, update tgbotapi.Update) {
	flag := false
	rows, err := db.Query("SELECT urg,date,level,info,price FROM asking WHERE idSolv = " + strconv.Itoa(hook.userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", hook.userID, err.Error())
	} else {
		for rows.Next() {
			var (
				urg   string
				date  string
				level string
				info  string
				price string

				urgPre string
			)
			rows.Scan(&urg, &date, &level, &info, &price)

			if urg == "quick" {
				urgPre = "Срочная"
			}
			if urg == "slow" {
				urgPre = "Не срочная"
			}

			htmlText := `<b>` + urgPre + `</b>
` + date + `
<b>Уровень</b> ` + level + `
<b>Информация</b> ` + info + `
<b>Цена</b> ` + price

			ask := tgbotapi.NewMessage(hook.chatID, htmlText)
			ask.ParseMode = tgbotapi.ModeHTML
			ask.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			flag = true
			bot.Send(ask)
		}
		if flag == false {
			msg := tgbotapi.NewMessage(hook.chatID, "У вас еще нет заявок")
			msg.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
			bot.Send(msg)
			return
		}
	}
}

// Основная ручка запроса
func webhookHandler(c *gin.Context) {
	defer c.Request.Body.Close()

	// Чтение запроса
	bytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Fatalf("[X] Could not read request. Reason: %s", err.Error())
		return
	}

	// Update
	var update tgbotapi.Update
	err = json.Unmarshal(bytes, &update)
	if err != nil {
		log.Fatalf("[X] Could not unmarshal updates. Reason: %s", err.Error())
		return
	}
	//
	//
	// Основная инфа об update
	hook := parseUpdate(update)
	hook.inTable = checkUserID(db, hook.userID)
	//
	// Логика старого пользователя
	if hook.inTable == true {
		hook.status = getText(db, "bot_user", hook.userID, "status")

		if strings.HasPrefix(hook.status, "reg") {
			logicReg(hook, update)
		}
		if strings.HasPrefix(hook.status, "ask") {
			logicAsk(hook, update)
		}
		if strings.HasPrefix(hook.status, "search") {
			switch hook.hasCallback {
			case true:
				if update.CallbackQuery.Data == "Закончить" {
					newStatus(db, hook.userID, "menu")
					menu := tgbotapi.NewMessage(hook.chatID, "Ok")
					menu.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
					bot.Send(menu)
				} else if update.CallbackQuery.Data == "Следующая" {
					logicSearch(hook, update)
				} else if update.CallbackQuery.Data == "Взять" {
					logicTake(hook, update)
					logicSearch(hook, update)
				}
			default:
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Нажимайте не кнопочки"))
			}
		}

		if hook.status == "menu" {
			if hook.hasText == false {
				menu := tgbotapi.NewMessage(hook.chatID, "Пожалуйста выбирете что то из меню")
				menu.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
				bot.Send(menu)
			} else {
				switch update.Message.Text {
				case "Мой профиль":
					userProfile(hook, update)
				case "Удалить профиль":
					deleteUser(db, hook.userID)
					bot.Send(tgbotapi.NewMessage(hook.chatID, "Ваш профиль удален, если хотите создать новый профиль введите /start"))
				case "Подать заявку":
					if getText(db, "bot_user", hook.userID, "role") == "solver" {
						menu := tgbotapi.NewMessage(hook.chatID, "Пожалуйста выбирете что то из меню")
						menu.ReplyMarkup = menuReply("solver")
						bot.Send(menu)
					} else {
						chooseUrg := tgbotapi.NewMessage(hook.chatID, "Эта заявка срочная или нет?")
						keyboard := tgbotapi.InlineKeyboardMarkup{}
						var row []tgbotapi.InlineKeyboardButton
						row = append(row, tgbotapi.NewInlineKeyboardButtonData("Срочная", "Срочная"))
						row = append(row, tgbotapi.NewInlineKeyboardButtonData("Не срочная", "Не срочная"))
						keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
						chooseUrg.ReplyMarkup = keyboard
						bot.Send(chooseUrg)

						newStatus(db, hook.userID, "ask1")
						newAsk(db, hook.userID, hook.updateID)

					}
				case "Мои заявки":
					userAsk(hook, update)
				case "Мои ответы на заявки":
					userSolv(hook, update)
				case "Поиск заявок":
					if getText(db, "bot_user", hook.userID, "role") == "asking" {
						menu := tgbotapi.NewMessage(hook.chatID, "Пожалуйста выбирете что то из меню")
						menu.ReplyMarkup = menuReply("solver")
						bot.Send(menu)
					} else {
						newStatus(db, hook.userID, "search")
						logicSearch(hook, update)
					}
				default:
					menu := tgbotapi.NewMessage(hook.chatID, "Пожалуйста выбирете что то из меню")
					menu.ReplyMarkup = menuReply(getText(db, "bot_user", hook.userID, "role"))
					bot.Send(menu)
				}
			}
		}

	}
	//
	// Логика нового пользователя
	if hook.inTable == false {
		if hook.hasText == false {
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста, начните с команды /start"))
		} else if update.Message.Command() == "start" {
			// Приветствие до регистрации
			htmlText := `Привет!. Этот бот решит твои учебные вопросы, с помощью других людей. Ты можешь сам стать одним из них. Но сперва, давай пройдем короткую регистрацию`
			msg := tgbotapi.NewMessage(hook.chatID, htmlText)
			msg.ParseMode = tgbotapi.ModeHTML
			bot.Send(msg)

			// Выбор роли
			chooseRole := tgbotapi.NewMessage(hook.chatID, "Кем ты хочешь быть?")
			keyboard := tgbotapi.InlineKeyboardMarkup{}
			var row []tgbotapi.InlineKeyboardButton
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Решателем", "Решателем"))
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Спрашивателем", "Спрашивателем"))
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
			chooseRole.ReplyMarkup = keyboard
			bot.Send(chooseRole)
			newID(db, hook.userID)

		} else {
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста, начните с команды /start"))
		}
	}

}

//
// telegram api
func initTelegram() {
	var err error

	// Рождение телеграм бота через botToken
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("[X] Could not create bot. Reason: %s", err.Error())
		return
	}

	// Даем боту урл для общения с сервером и получения обновлений
	url := baseURL + bot.Token
	_, err = bot.SetWebhook(tgbotapi.NewWebhook(url))
	if err != nil {
		log.Fatalf("[X] Could not set webhook to bot settings. Reason: %s", err.Error())
	}
}

//
// MAIN
//
func main() {
	var err error

	// Получение порта
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("[X] $PORT must be set")
	} else {
		log.Printf("[OK] Get PORT = %s", port)
	}

	// gin router запуск
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// DB проверка соединения
	/*
		db, err = sql.Open("postgres", dbURI)
		if err != nil {
			log.Fatalf("[X] Could not connect to DB. Reason: %s", err.Error())
		} else {
			log.Printf("[OK] Connect DB")
		}
		defer db.Close()
	*/

	// Таблица учителей на всякий случай создается заново или ничего не делается

	//truncateTable(db, "bot_user")				// Удаление всех записей != Удаление таблицы
	//truncateTable(db, "asking")
	//
	//dropTable(db, "bot_user") // Удаление таблицы
	//dropTable(db, "asking")
	//
	//time.Sleep(2 * time.Second) // Спячка на 2 секунд
	//
	createTable(db, "bot_user")
	createTable(db, "asking")

	// telegram api
	initTelegram()
	router.POST("/"+bot.Token, webhookHandler) // Хуки api телеграма, обход перезагрузки heroku

	err = router.Run(":" + port) // Слушаем запросы
	if err != nil {
		log.Fatalf("[X] Could not run router. Reason: %s", err.Error())
	}
}

// НИЖЕ ИДУТ ВСЕ ФУНКЦИИ С DB

// Создание таблицы
//
// bot_user(id INT PRIMARY KEY,role TEXT,name TEXT,surname TEXT,level TEXT,img TEXT,status TEXT,lastask INT)
//
// asking(idUser INT,id INT PRIMARY KEY,idSolv INT,urg TEXT,date TEXT,level TEXT,info TEXT,price TEXT)
func createTable(db *sql.DB, name string) {
	if name == "bot_user" {
		_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (id INT PRIMARY KEY,role TEXT,name TEXT,surname TEXT,level TEXT,img TEXT,status TEXT,lastask INT);")
		if err != nil {
			log.Fatalf("[X] Could not create %s table. Reason: %s", name, err.Error())
		} else {
			log.Printf("[OK] Create %s table", name)
		}
	} else if name == "asking" {
		_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (idUser INT,id INT PRIMARY KEY,idSolv INT,urg TEXT,date TEXT,level TEXT,info TEXT,price TEXT);")
		if err != nil {
			log.Fatalf("[X] Could not create %s table. Reason: %s", name, err.Error())
		} else {
			log.Printf("[OK] Create %s table", name)
		}
	} else {
		log.Printf("[ERR] Wrong %s table DB", name)
	}
}

// Удаление таблицы
func dropTable(db *sql.DB, name string) {
	_, err := db.Exec("DROP TABLE " + name + ";")
	if err != nil {
		log.Fatalf("[X] Could not drop %s table. Reason: %s", name, err.Error())
	} else {
		log.Printf("[OK] Drop %s table", name)
	}
}

// Сброс таблицы
func truncateTable(db *sql.DB, name string) {
	_, err := db.Exec("TRUNCATE TABLE " + name + ";")
	if err != nil {
		log.Fatalf("[X] Could not truncate %s table. Reason: %s", name, err.Error())
	} else {
		log.Printf("[OK] Truncate %s table", name)
	}
}

// Удаление id
func deleteUser(db *sql.DB, userID int) {
	_, err := db.Exec("DELETE FROM bot_user WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not delete %d from bot_user table. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] Delete %d", userID)
	}
	_, err = db.Exec("DELETE FROM asking WHERE idUser = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not delete %d from asking table. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] Delete ask %d", userID)
	}
}

// Проверяем есть ли id в таблице bot_user
//
// have connection
func checkUserID(db *sql.DB, userID int) bool {
	rows, err := db.Query("SELECT id FROM bot_user WHERE id = " + strconv.Itoa(userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select id. Reason: %s", err.Error())
	} else {
		for rows.Next() {
			return true
		}
		return false
	}
	return false
}

// Изменение статуса
func newStatus(db *sql.DB, userID int, status string) {
	_, err := db.Exec("UPDATE bot_user SET status = '" + status + "' WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not update status to %d. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] %d update status to %s", userID, status)
	}
}

// Добавление нового пользователя
func newID(db *sql.DB, userID int) {
	_, err := db.Exec("INSERT INTO bot_user (id,status) VALUES (" + strconv.Itoa(userID) + ", 'reg1');")
	if err != nil {
		log.Fatalf("[X] Could not insert newID. Reason: %s", err.Error())
	} else {
		log.Printf("[OK] New user %d", userID)
	}
}

// Добавление новой заявки
func newAsk(db *sql.DB, userID int, askID int) {
	_, err := db.Exec("INSERT INTO asking (idUser,id,idSolv) VALUES (" + strconv.Itoa(userID) + ", " + strconv.Itoa(askID) + ", 0);")
	if err != nil {
		log.Fatalf("[X] Could not create new ask %d. Reason: %s", askID, err.Error())
	} else {
		log.Printf("[OK] New ask %d", askID)
	}

	_, err = db.Exec("UPDATE bot_user SET lastask = " + strconv.Itoa(askID) + " WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not update lastask user %d. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] %d update lastask to %d", userID, askID)
	}
}

// В table для userID в колонку column поместить текст value
// Достать column из table для userID
func setText(db *sql.DB, table string, userID int, column string, value string) {
	_, err := db.Exec("UPDATE " + table + " SET " + column + " = '" + value + "' WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could no update %d. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] User %d update", userID)
	}
}
func getText(db *sql.DB, table string, userID int, column string) (value string) {
	rows, err := db.Query("SELECT " + column + " FROM " + table + " WHERE id = " + strconv.Itoa(userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", userID, err.Error())
	} else {
		for rows.Next() {
			rows.Scan(&value)
		}
	}
	return value
}
func setInt(db *sql.DB, table string, userID int, column string, value int) {
	_, err := db.Exec("UPDATE " + table + " SET " + column + " = " + strconv.Itoa(value) + " WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not update %d. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] User %d update", userID)
	}
}
func getInt(db *sql.DB, table string, userID int, column string) (value int) {
	rows, err := db.Query("SELECT " + column + " FROM " + table + " WHERE id = " + strconv.Itoa(userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", userID, err.Error())
	} else {
		for rows.Next() {
			rows.Scan(&value)
		}
	}
	return value
}
