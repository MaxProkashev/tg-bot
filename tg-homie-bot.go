package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

// menu
func menuBot() interface{} {
	return tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Мой профиль"),
				tgbotapi.NewKeyboardButton("Удалить профиль"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Мои ответы"),
				tgbotapi.NewKeyboardButton("Подать заявку"),
				tgbotapi.NewKeyboardButton("Мои заявки"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Поиск заявок"),
			),
		},
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
	}
}

// Логика регистрации пользователя
func logicReg(hook hookConfig, update tgbotapi.Update) {
	switch hook.status {
	case "reg1": // name
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "reg2")
			setText(db, "bot_user", hook.userID, "name", update.Message.Text)
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Ok, Ваша фамилия?"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите текст 📝"))
		}
	case "reg2": // surname
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "reg3")
			setText(db, "bot_user", hook.userID, "surname", update.Message.Text)
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Ok, давайте добавим фотографию 📷 в профиль"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите текст 📝"))
		}
	case "reg3": // img
		switch hook.hasPhoto {
		case true:
			newStatus(db, hook.userID, "reg4")
			setText(db, "bot_user", hook.userID, "img", (*update.Message.Photo)[0].FileID)

			msg := tgbotapi.NewMessage(hook.chatID, "Ok, Вы учитесь?")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("В школе", "В школе"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("В колледже", "В колледже"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("В университете", "В университете"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Нет", "Нет"),
				),
			)
			bot.Send(msg)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Фотографию 📷?"))
		}
	case "reg4": //study
		switch hook.hasCallback {
		case true:
			switch update.CallbackQuery.Data {
			case "В школе":
				newStatus(db, hook.userID, "reg4school")
				msg := tgbotapi.NewMessage(hook.chatID, "Какой класс?")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Начальная школа", "Начальная школа"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("5", "5"),
						tgbotapi.NewInlineKeyboardButtonData("6", "6"),
						tgbotapi.NewInlineKeyboardButtonData("7", "7"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("8", "8"),
						tgbotapi.NewInlineKeyboardButtonData("9", "9"),
						tgbotapi.NewInlineKeyboardButtonData("10", "10"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("11", "11"),
					),
				)
				bot.Send(msg)
			case "В колледже":
				newStatus(db, hook.userID, "reg4college")
				msg := tgbotapi.NewMessage(hook.chatID, "Какой курс?")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("1", "1"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("2", "2"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("3", "3"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("4", "4"),
					),
				)
				bot.Send(msg)
			case "В университете":
				newStatus(db, hook.userID, "reg4univer")
				msg := tgbotapi.NewMessage(hook.chatID, "Какой курс?")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("1", "1"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("2", "2"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("3", "3"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("4", "4"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("5", "5"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("6", "6"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Магистратура", "Магистратура"),
					),
				)
				bot.Send(msg)
			case "Нет":
				newStatus(db, hook.userID, "reg5")
				setText(db, "bot_user", hook.userID, "study", "не учится")
				msg := tgbotapi.NewMessage(hook.chatID, "Ok, Вы работаете?")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Да", "Да"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Нет", "Нет"),
					),
				)
				bot.Send(msg)
			default:
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
		}
	case "reg4school":
		switch hook.hasCallback {
		case true:
			switch update.CallbackQuery.Data {
			case "Начальная школа":
				newStatus(db, hook.userID, "reg4schoolnum")
				setText(db, "bot_user", hook.userID, "temp", "начальные классы")
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите номер школы"))
			default:
				newStatus(db, hook.userID, "reg4schoolnum")
				setText(db, "bot_user", hook.userID, "temp", update.CallbackQuery.Data+" класс")
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите номер школы"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
		}
	case "reg4schoolnum":
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "reg5")
			temp := getText(db, "bot_user", hook.userID, "temp")
			setText(db, "bot_user", hook.userID, "study", "школа "+update.Message.Text+" "+temp)

			msg := tgbotapi.NewMessage(hook.chatID, "Ok, Вы работаете?")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Да", "Да"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Нет", "Нет"),
				),
			)
			bot.Send(msg)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите текст 📝"))
		}
	case "reg4college":
		switch hook.hasCallback {
		case true:
			newStatus(db, hook.userID, "reg4collegenum")
			setText(db, "bot_user", hook.userID, "temp", update.CallbackQuery.Data+" курс")
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите название колледжа"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
		}
	case "reg4collegenum":
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "reg5")
			temp := getText(db, "bot_user", hook.userID, "temp")
			setText(db, "bot_user", hook.userID, "study", update.Message.Text+" "+temp)

			msg := tgbotapi.NewMessage(hook.chatID, "Ok, Вы работаете?")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Да", "Да"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Нет", "Нет"),
				),
			)
			bot.Send(msg)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите текст 📝"))
		}
	case "reg4univer":
		switch hook.hasCallback {
		case true:
			switch update.CallbackQuery.Data {
			case "Магистратура":
				newStatus(db, hook.userID, "reg4univernum")
				setText(db, "bot_user", hook.userID, "temp", "магистратура")
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Название университета?"))
			default:
				newStatus(db, hook.userID, "reg4univernum")
				setText(db, "bot_user", hook.userID, "temp", update.CallbackQuery.Data+" курс")
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Название университета?"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
		}
	case "reg4univernum":
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "reg5")
			temp := getText(db, "bot_user", hook.userID, "temp")
			setText(db, "bot_user", hook.userID, "study", update.Message.Text+" "+temp)

			msg := tgbotapi.NewMessage(hook.chatID, "Ok, Вы работаете?")
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Да", "Да"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Нет", "Нет"),
				),
			)
			bot.Send(msg)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите текст 📝"))
		}
	case "reg5": //work
		switch hook.hasCallback {
		case true:
			switch update.CallbackQuery.Data {
			case "Да":
				newStatus(db, hook.userID, "reg5work")
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Кем Вы работаете?"))
			case "Нет":
				newStatus(db, hook.userID, "menu")
				setText(db, "bot_user", hook.userID, "work", "не работает")

				menu := tgbotapi.NewMessage(hook.chatID, "Ура! Вы в системе 🌐")
				menu.ReplyMarkup = menuBot()
				bot.Send(menu)
			default:
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
			}
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Выеберете один из вариантов ответа"))
		}
	case "reg5work":
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "menu")
			setText(db, "bot_user", hook.userID, "work", update.Message.Text)

			menu := tgbotapi.NewMessage(hook.chatID, "Ура! Вы в системе 🌐")
			menu.ReplyMarkup = menuBot()
			bot.Send(menu)
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Введите текст 📝"))
		}
	}
}

// Логика регистрации заявки
func logicAsk(hook hookConfig, update tgbotapi.Update) {
	id := getInt(db, "bot_user", hook.userID, "lastask")
	switch hook.status {
	case "ask1": // theme
		switch hook.hasCallback {
		case true:
			newStatus(db, hook.userID, "ask2")
			setText(db, "asking", id, "theme", update.CallbackQuery.Data)
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Немного опишите вашу заявку"))
		default:
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Выберете один из предметов"))
		}
	case "ask2": // info
		switch hook.hasText {
		case true:
			newStatus(db, hook.userID, "menu")
			setText(db, "asking", id, "info", update.Message.Text)

			time := time.Unix(int64(update.Message.Date), 0).Add(3 * time.Hour)
			setText(db, "asking", id, "date", strconv.Itoa(time.Day())+" "+time.Month().String()+" "+strconv.Itoa(time.Hour())+":"+strconv.Itoa(time.Minute()))

			menu := tgbotapi.NewMessage(hook.chatID, "Заявка принята")
			menu.ReplyMarkup = menuBot()
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
	rows, err := db.Query("SELECT iduser,id,idsolv,date,theme,info FROM asking WHERE id <> " + strconv.Itoa(id) + " AND idsolv = 0 AND iduser <> " + strconv.Itoa(hook.userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", hook.userID, err.Error())
	} else {
		for rows.Next() {
			var (
				iduser int
				id     int
				idsolv int
				date   string
				theme  string
				info   string
			)
			rows.Scan(&iduser, &id, &idsolv, &date, &theme, &info)
			setInt(db, "bot_user", hook.userID, "lastask", id)

			name := getText(db, "bot_user", iduser, "name")
			surname := getText(db, "bot_user", iduser, "surname")
			study := getText(db, "bot_user", iduser, "study")
			work := getText(db, "bot_user", iduser, "work")

			text := fmt.Sprintf(
				"*Информация о том кто подал заявку:*\n"+
					"%s %s\n"+
					"*Место учебы:* %s\n"+
					"*Место работы:* %s\n\n"+
					"*Информация о заявке:*\n"+
					"*Дата подачи заявки:* %s\n"+
					"*Тема:* %s\n"+
					"*Описание:* %s\n",
				name,
				surname,
				study,
				work,
				date,
				theme,
				info,
			)

			NextEnd := tgbotapi.NewMessage(hook.chatID, text)
			NextEnd.ParseMode = tgbotapi.ModeMarkdown
			keyboard := tgbotapi.InlineKeyboardMarkup{}
			var row []tgbotapi.InlineKeyboardButton
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Взять", "Взять"))
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Закончить", "Закончить"))
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Следующая", "Следующая"))
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
			NextEnd.ReplyMarkup = keyboard
			bot.Send(NextEnd)
			flag = true
			return
		}
		if flag == false {
			newStatus(db, hook.userID, "menu")
			msg := tgbotapi.NewMessage(hook.chatID, "Свободные заявки кончились")
			msg.ReplyMarkup = menuBot()
			bot.Send(msg)
			return
		}
	}
}

func logicTake(hook hookConfig, update tgbotapi.Update) {
	id := getInt(db, "bot_user", hook.userID, "lastask")
	setInt(db, "asking", id, "idsolv", hook.userID)
}

// Профиль пользователя
func userProfile(hook hookConfig, update tgbotapi.Update) {
	name := getText(db, "bot_user", hook.userID, "name")
	surname := getText(db, "bot_user", hook.userID, "surname")
	study := getText(db, "bot_user", hook.userID, "study")
	work := getText(db, "bot_user", hook.userID, "work")

	text := fmt.Sprintf(
		"*%s %s*\n"+
			"*Место учебы:* %s\n"+
			"*Место работы:* %s\n",
		name, surname,
		study,
		work,
	)
	menu := tgbotapi.NewMessage(hook.chatID, text)
	menu.ReplyMarkup = menuBot()
	menu.ParseMode = tgbotapi.ModeMarkdown

	img := tgbotapi.NewPhotoShare(hook.chatID, getText(db, "bot_user", hook.userID, "img"))

	bot.Send(img)
	bot.Send(menu)
}

// Заявки пользователя
func userAsk(hook hookConfig, update tgbotapi.Update) {
	//name := getText(db, "bot_user", hook.userID, "name")
	//surname := getText(db, "bot_user", hook.userID, "surname")
	//study := getText(db, "bot_user", hook.userID, "study")
	//work := getText(db, "bot_user", hook.userID, "work")

	flag := false
	rows, err := db.Query("SELECT idsolv,date,theme,info FROM asking WHERE iduser = " + strconv.Itoa(hook.userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", hook.userID, err.Error())
	} else {
		for rows.Next() {
			var (
				idsolv int
				date   string
				theme  string
				info   string
			)
			rows.Scan(&idsolv, &date, &theme, &info)
			text := fmt.Sprintf(
				"*Дата подачи заявки:* %s\n"+
					"*Тема:* %s\n"+
					"*Описание:* %s\n",
				date,
				theme,
				info,
			)
			if idsolv != 0 {
				text = text + "\n*Заявку взял:* " + strconv.Itoa(idsolv)
			} else {
				text = text + "\n*Заявку еще никто не взял*"
			}

			ask := tgbotapi.NewMessage(hook.chatID, text)
			ask.ParseMode = tgbotapi.ModeMarkdown
			ask.ReplyMarkup = menuBot()
			flag = true
			bot.Send(ask)
		}
		if flag == false {
			msg := tgbotapi.NewMessage(hook.chatID, "У вас еще нет заявок")
			msg.ReplyMarkup = menuBot()
			bot.Send(msg)
			return
		}
	}
}

// Заявки которые взял пользователь
func userSolv(hook hookConfig, update tgbotapi.Update) {
	flag := false
	rows, err := db.Query("SELECT iduser,id,date,theme,info FROM asking WHERE idsolv = " + strconv.Itoa(hook.userID) + ";")
	defer rows.Close()
	if err != nil {
		log.Fatalf("[X] Could not select %d. Reason: %s", hook.userID, err.Error())
	} else {
		for rows.Next() {
			var (
				iduser int
				id     int
				idsolv int
				date   string
				theme  string
				info   string
			)
			rows.Scan(&iduser, &id, &idsolv, &date, &theme, &info)

			name := getText(db, "bot_user", iduser, "name")
			surname := getText(db, "bot_user", iduser, "surname")
			study := getText(db, "bot_user", iduser, "study")
			work := getText(db, "bot_user", iduser, "work")

			text := fmt.Sprintf(
				"*Информация о том кто подал заявку*\n"+
					"%s %s\n"+
					"*Место учебы:* %s\n"+
					"*Место работы:* %s\n\n"+
					"*Информация о заявке:*\n"+
					"*Дата подачи заявки:* %s\n"+
					"*Тема:* %s\n"+
					"*Описание:* %s\n",
				name,
				surname,
				study,
				work,
				date,
				theme,
				info,
			)

			ask := tgbotapi.NewMessage(hook.chatID, text)
			ask.ParseMode = tgbotapi.ModeMarkdown
			ask.ReplyMarkup = menuBot()
			flag = true
			bot.Send(ask)
		}
		if flag == false {
			msg := tgbotapi.NewMessage(hook.chatID, "Вы не взяли ни одной заявки")
			msg.ReplyMarkup = menuBot()
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
					menu.ReplyMarkup = menuBot()
					bot.Send(menu)
				} else if update.CallbackQuery.Data == "Следующая" {
					logicSearch(hook, update)
				} else if update.CallbackQuery.Data == "Взять" {
					logicTake(hook, update)
					bot.Send(tgbotapi.NewMessage(hook.chatID, "Заявка теперь ваша"))
					logicSearch(hook, update)
				}
			default:
				bot.Send(tgbotapi.NewMessage(hook.chatID, "Выберете один из вариантов ответа"))
			}
		}

		if hook.status == "menu" {
			if hook.hasText == false {
				menu := tgbotapi.NewMessage(hook.chatID, "Пожалуйста выбирете что то из меню")
				menu.ReplyMarkup = menuBot()
				bot.Send(menu)
			} else {
				switch update.Message.Text {
				case "Мой профиль":
					userProfile(hook, update)
				case "Удалить профиль":
					deleteUser(db, hook.userID)
					bot.Send(tgbotapi.NewMessage(hook.chatID, "Ваш профиль удален, если хотите создать новый профиль введите /start"))
				case "Подать заявку":
					choosetheme := tgbotapi.NewMessage(hook.chatID, "Выберете тему вашей заявки")
					choosetheme.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("Математика", "Математика"),
						),
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("Физика", "Физика"),
						),
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("Русский язык", "Русский язык"),
						),
					)
					bot.Send(choosetheme)
					newStatus(db, hook.userID, "ask1")
					newAsk(db, hook.userID, hook.updateID)
				case "Мои заявки":
					userAsk(hook, update)
				case "Мои ответы":
					userSolv(hook, update)
				case "Поиск заявок":
					newStatus(db, hook.userID, "search")
					logicSearch(hook, update)
				default:
					menu := tgbotapi.NewMessage(hook.chatID, "Пожалуйста выбирете что то из меню")
					menu.ReplyMarkup = menuBot()
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
			htmlText := "(Придумать приветствие)"
			msg := tgbotapi.NewMessage(hook.chatID, htmlText)
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)

			// new id
			newID(db, hook.userID)
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Как Вас зовут?"))

		} else {
			bot.Send(tgbotapi.NewMessage(hook.chatID, "Пожалуйста, начните с команды /start"))
		}
	}

}

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
// bot_user(id INT PRIMARY KEY,name TEXT,surname TEXT,img TEXT,study TEXT,work TEXT,status TEXT,lastask INT,temp TEXT)
//
// asking(iduser INT,id INT PRIMARY KEY,idsolv INT,date TEXT,theme TEXT,info TEXT)
func createTable(db *sql.DB, name string) {
	if name == "bot_user" {
		_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (id INT PRIMARY KEY,name TEXT,surname TEXT,img TEXT,study TEXT,work TEXT,status TEXT,lastask INT,temp TEXT);")
		if err != nil {
			log.Fatalf("[X] Could not create %s table. Reason: %s", name, err.Error())
		} else {
			log.Printf("[OK] Create %s table", name)
		}
	} else if name == "asking" {
		_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + name + " (iduser INT,id INT PRIMARY KEY,idsolv INT,date TEXT,theme TEXT,info TEXT);")
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

// Работа с таблицей int, text
// В table для userID в колонку column поместить текст value
// Достать column из table для userID
func setText(db *sql.DB, table string, userID int, column string, value string) {
	_, err := db.Exec("UPDATE " + table + " SET " + column + " = '" + value + "' WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could no update %d. Reason: %s", userID, err.Error())
	} /*else {
		log.Printf("[OK] User %d update", userID)
	}*/
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
	} /*else {
		log.Printf("[OK] User %d update", userID)
	}*/
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

// Логика таблицы
//
// Проверяем есть ли id в таблице bot_user
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
	} /*else {
		log.Printf("[OK] %d update status to %s", userID, status)
	}*/
}

// Добавление нового пользователя
func newID(db *sql.DB, userID int) {
	_, err := db.Exec("INSERT INTO bot_user (id,status,lastask) VALUES (" + strconv.Itoa(userID) + ", 'reg1', 0);")
	if err != nil {
		log.Fatalf("[X] Could not insert newID. Reason: %s", err.Error())
	} else {
		log.Printf("[OK] New user %d", userID)
	}
}

// Удаление пользователя
func deleteUser(db *sql.DB, userID int) {
	_, err := db.Exec("DELETE FROM bot_user WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not delete %d from bot_user table. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] Delete user %d", userID)
	}
	_, err = db.Exec("DELETE FROM asking WHERE iduser = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not delete %d from asking table. Reason: %s", userID, err.Error())
	} else {
		log.Printf("[OK] Delete ask %d", userID)
	}
}

// Добавление новой заявки
func newAsk(db *sql.DB, userID int, askID int) {
	_, err := db.Exec("INSERT INTO asking (iduser,id,idsolv) VALUES (" + strconv.Itoa(userID) + ", " + strconv.Itoa(askID) + ", 0);")
	if err != nil {
		log.Fatalf("[X] Could not create new ask %d. Reason: %s", askID, err.Error())
	} else {
		log.Printf("[OK] New ask %d", askID)
	}

	_, err = db.Exec("UPDATE bot_user SET lastask = " + strconv.Itoa(askID) + " WHERE id = " + strconv.Itoa(userID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not update lastask user %d. Reason: %s", userID, err.Error())
	} /*else {
		log.Printf("[OK] %d update lastask to %d", userID, askID)
	}*/
}

// Удаление заявки
func deleteAsk(db *sql.DB, askID int) {
	_, err := db.Exec("DELETE FROM asking WHERE id = " + strconv.Itoa(askID) + ";")
	if err != nil {
		log.Fatalf("[X] Could not delete %d from asking table. Reason: %s", askID, err.Error())
	} else {
		log.Printf("[OK] Delete ask %d", askID)
	}
}
