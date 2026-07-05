package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rishit-kadha/peak/internal/postgres"
	"github.com/rishit-kadha/peak/internal/db"
	"sync"
	"fmt"
)

type command struct{
	name string
	description string
	callback func(ctx context.Context, b *bot.Bot, update *models.Update)
}
type userState struct{

}
type stateKey struct {
    ChatID int64
    UserID int64
}

type App struct {
    queries  db.Queries
    commands map[string]command
    mu       sync.RWMutex
    state    map[stateKey]userState
}

func main() {
	// A context to stop go routines and stuff when os.Interrupt i.e. process is exited or something
	// defer cancel() cancels the context before returning
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// DATABASE CONNECTION
	pool , err := postgres.New()
	if err != nil {
		log.Fatalf("POSTGRES CONNECTION FAILED : %v", err)
	}
	defer pool.Close()
	//sqlc connection
	q := db.New(pool)


	// APP STATE 

	// initialize state
	app := &App{}
	app.commands = map[string]command{
		"/help" : {
			name : "/help",
			description : "List Commands",
			callback : app.helpHandler,
		},
		"/start" : {
			name : "/start",
			description : "Start Menu / Setup For New Users",
			callback : app.startHandler,
		},
	}
	app.queries = *q

	//TELEGRAM SETUP

	opts := []bot.Option{
		bot.WithDefaultHandler(app.defaultHandler),
	}

	b, err := bot.New(os.Getenv("TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		log.Fatalf("FAILED TO CREATE TELEGRAM BOT: %v ", err)
	}
	// Automated Registration
	for pattern, cmd := range app.commands {
		b.RegisterHandler(bot.HandlerTypeMessageText, pattern, bot.MatchTypeExact, cmd.callback)
	}
	b.Start(ctx)
}

func (a *App) getState(chatID, userID int64) userState {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.state[stateKey{ChatID: chatID, UserID: userID}]
}

func (a *App) setState(chatID, userID int64, s userState) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.state[stateKey{ChatID: chatID, UserID: userID}] = s
}

func (app *App) defaultHandler(ctx context.Context , b *bot.Bot , update *models.Update){
	b.SendMessage( ctx , &bot.SendMessageParams{
		ChatID : update.Message.Chat.ID ,
		Text : "Sorry I didn't get that , Try using /start or /help" ,
	})
}

func (app *App) startHandler(ctx context.Context , b *bot.Bot , update *models.Update){
	//check if user is registered 
	_, err := app.queries.GetUserByTelegramUserID(ctx ,update.Message.From.ID)
	if err == nil{
		// user exists
		b.SendMessage(ctx , &bot.SendMessageParams{
		ChatID : update.Message.Chat.ID ,
		Text : "Welcome Back :3 , use /help to List commands",
		})
	}else{
		//user does not exist
		_ , err := app.queries.CreateUser(ctx ,update.Message.From.ID)
		if err != nil{
			// error creating user 
			log.Printf("CreateUser failed: %v", err)
			b.SendMessage(ctx , &bot.SendMessageParams{
			ChatID : update.Message.Chat.ID ,
			Text : "Unable to Register , Try again later",
			})
		}else{
			b.SendMessage(ctx , &bot.SendMessageParams{
			ChatID : update.Message.Chat.ID ,
			Text : "Welcome to Peak , Registration succesful , you can start you journey today , try /help to learn more",
			})
		}

	}
}

func (app *App) helpHandler(ctx context.Context , b *bot.Bot , update *models.Update){
	message := "LIST OF COMMANDS : \n\n" 
	i := 1
	for _ , cmdStruct := range app.commands{
		message = message + fmt.Sprintf("%v. %v : %v \n",i,cmdStruct.name,cmdStruct.description)  
		i++
	}
	b.SendMessage(ctx , &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID ,
		Text : message,
	})
}