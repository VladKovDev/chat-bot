package conversation

type BotResponse struct {
	Text string
	Buttons [][]Button
	ShouldCreateTicket bool
	TicketCategory string
}

type Button struct {
	Text string
	Event Event
}

func resolvedButtons() [][]Button {
	return [][]Button{
		{
			{Text: "Да", Event: EventResolved},
			{Text: "Нет", Event: EventNotResolved},
		},
	}
}

func categoryButtons() [][]Button {
	return [][]Button{
		{
			{Text: "Вход / пароль", Event: EventCategorySelected},
			{Text: "Платежи", Event: EventCategorySelected},
			{Text: "Другое", Event: EventCategorySelected},
		},
		{
			{Text: "Запрос оператора", Event: EventRequestOperator},
		},
	}
}