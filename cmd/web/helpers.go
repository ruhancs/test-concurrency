package main

func (app *Config) sendEmail(msg Message) {
	app.Wait.Add(1)
	//envia a msg para o canal que esta sendo escutado em mailer e envia o email
	app.Mailer.MailerChan <- msg
}