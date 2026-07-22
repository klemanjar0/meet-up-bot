// Package i18n provides a tiny, dependency-free translator for the bot's
// user-facing strings, currently in English and Russian. Each message is a Key
// mapped to per-locale templates; templates may contain fmt verbs which T fills
// in with the supplied arguments (keep the argument order identical across
// locales, or use explicit indexes like %[1]s).
package i18n

import "fmt"

// Locale is a supported language code.
type Locale string

const (
	En Locale = "en"
	Ru Locale = "ru"
)

// Parse maps a stored string to a Locale, defaulting to English.
func Parse(s string) Locale {
	if Locale(s) == Ru {
		return Ru
	}
	return En
}

// Key identifies a translatable message.
type Key int

const (
	KeyHelp Key = iota
	KeyCancelled
	KeyErrGeneric
	KeyErrLoadLobbies
	KeyNoLobbies
	KeyErrLoadYourLobbies
	KeyNoYourLobbies
	KeyDidntGet

	KeyBtnJoin
	KeyBtnRequestJoin
	KeyBtnManageMembers
	KeyBtnEdit

	// Create wizard.
	KeyCreateStart
	KeyNameEmpty
	KeyAskDescription
	KeyAskCountry
	KeyCountryEmpty
	KeyAskCity
	KeyCityEmpty
	KeyAskAddress
	KeyAskTime
	KeyTimeBad
	KeyAskLink
	KeyAskVisibility
	KeyBtnPublic
	KeyBtnPrivateApproval
	KeyPickButtons
	KeyWizardExpired
	KeyUnknownOption
	KeyErrSaveLobby
	KeyLobbyCreated
	KeyCreatedPublicHint
	KeyCreatedPrivateHint

	// Join / approval.
	KeyAlreadyIn
	KeyStillPending
	KeyWasDeclined
	KeyBanned
	KeyJoinedToast
	KeyJoinedMsg
	KeyChatLine
	KeyRequestSentToast
	KeyRequestSentMsg
	KeyAdminNewRequest
	KeyBtnApprove
	KeyBtnReject
	KeyLobbyGone
	KeyAdminOnly
	KeyApprovedToast
	KeyApprovedEdit
	KeyApprovedMsg
	KeyRejectedToast
	KeyRejectedEdit
	KeyRejectedMsg
	KeyMalformed
	KeyPendingRequest

	// Members management.
	KeyNoMembersYet
	KeyBtnRemove
	KeyBtnBan
	KeyErrLoadMembers
	KeyCantRemoveSelf
	KeyRemovedToast
	KeyRemovedEdit
	KeyRemovedMsg
	KeyCantBanSelf
	KeyBannedToast
	KeyBannedEdit
	KeyBannedMsg

	// Edit lobby.
	KeyEditMenu
	KeyBtnEditName
	KeyBtnEditDescription
	KeyBtnEditCountry
	KeyBtnEditCity
	KeyBtnEditAddress
	KeyBtnEditTime
	KeyBtnEditLink
	KeyBtnMakePublic
	KeyBtnMakePrivate
	KeyEditAskName
	KeyEditAskDescription
	KeyEditAskCountry
	KeyEditAskCity
	KeyEditAskAddress
	KeyEditAskTime
	KeyEditAskLink
	KeyEditSaved
	KeyEditNotify
	KeyEditExpired

	// Settings / locale.
	KeySettingsMenu
	KeyBtnSetLang
	KeyBtnSetTimezone
	KeyBtnSetCity
	KeyBtnSetTimeFilter
	KeySettingsPrompt
	KeyBtnEnglish
	KeyBtnRussian
	KeyLocaleSet
	KeyAskTimezone
	KeyBadTimezone
	KeyTimezoneSet
	KeyAskCitySetting
	KeyCitySet
	KeyCityCleared
	KeyTimeFilterPrompt
	KeyBtnFilterDay
	KeyBtnFilterWeek
	KeyBtnFilterMonth
	KeyBtnFilterAll
	KeyTimeFilterSet
	KeyNotSet

	// Lobby list / pagination.
	KeyBtnPrev
	KeyBtnNext
	KeyPageLabel
	KeyLobbiesFilterNote

	// Lobby rendering.
	KeyLobbyPublic
	KeyLobbyPrivate
	KeyJoinedCount

	// My lobbies / details.
	KeyRoleAdmin
	KeyRoleMember
	KeyBtnDetails
	KeyNoMyLobbies
	KeyNotAMember
	KeyStatusJoined
	KeyStatusPending

	// Delete / leave.
	KeyBtnDeleteLobby
	KeyBtnLeave
	KeyBtnConfirmDelete
	KeyBtnCancel
	KeyDeleteConfirm
	KeyDeletedToast
	KeyDeletedEdit
	KeyDeletedNotify
	KeyLeftToast
	KeyLeftEdit

	// Invite links.
	KeyBtnInvite
	KeyInviteText
)

type entry struct{ en, ru string }

var messages = map[Key]entry{
	KeyHelp: {
		en: "👋 <b>Meet-up bot</b>\n\nCreate and join event lobbies right here in Telegram.\n\n<b>Commands</b>\n/create — set up a new lobby\n/lobbies — browse lobbies you can join\n/mylobbies — manage lobbies you created\n/settings — change your language\n/cancel — abort the current action\n/help — show this message",
		ru: "👋 <b>Meet-up бот</b>\n\nСоздавайте лобби мероприятий и присоединяйтесь к ним прямо в Telegram.\n\n<b>Команды</b>\n/create — создать новое лобби\n/lobbies — список лобби для участия\n/mylobbies — управление вашими лобби\n/settings — изменить язык\n/cancel — отменить текущее действие\n/help — показать это сообщение",
	},
	KeyCancelled:          {en: "Cancelled. Nothing was saved.", ru: "Отменено. Ничего не сохранено."},
	KeyErrGeneric:         {en: "Something went wrong, please try again.", ru: "Что-то пошло не так, попробуйте ещё раз."},
	KeyErrLoadLobbies:     {en: "Sorry, could not load lobbies right now.", ru: "Не удалось загрузить лобби."},
	KeyNoLobbies:          {en: "No lobbies yet. Be the first — /create one!", ru: "Пока нет лобби. Создайте первое — /create!"},
	KeyErrLoadYourLobbies: {en: "Sorry, could not load your lobbies right now.", ru: "Не удалось загрузить ваши лобби."},
	KeyNoYourLobbies:      {en: "You haven't created any lobbies yet. Use /create.", ru: "Вы ещё не создали ни одного лобби. Используйте /create."},
	KeyDidntGet:           {en: "I didn't get that. Try /help to see what I can do.", ru: "Не понял. Напишите /help, чтобы увидеть возможности."},

	KeyBtnJoin:          {en: "✅ Join", ru: "✅ Присоединиться"},
	KeyBtnRequestJoin:   {en: "🔒 Request to join", ru: "🔒 Запросить участие"},
	KeyBtnManageMembers: {en: "👥 Manage members", ru: "👥 Участники"},
	KeyBtnEdit:          {en: "✏️ Edit", ru: "✏️ Редактировать"},

	KeyCreateStart:        {en: "Let's create a lobby! 🎉\n\nWhat's the <b>name</b> of your event?\n\n(Send /cancel any time to abort.)", ru: "Создаём лобби! 🎉\n\nКак называется ваше мероприятие? (<b>название</b>)\n\n(Отправьте /cancel, чтобы отменить.)"},
	KeyNameEmpty:          {en: "The name can't be empty. What's the event called?", ru: "Название не может быть пустым. Как называется мероприятие?"},
	KeyAskDescription:     {en: "Add a <b>description</b>, or send <code>-</code> to skip.", ru: "Добавьте <b>описание</b> или отправьте <code>-</code>, чтобы пропустить."},
	KeyAskCountry:         {en: "Which <b>country</b> is it in?", ru: "В какой <b>стране</b> оно пройдёт?"},
	KeyCountryEmpty:       {en: "Please tell me the country.", ru: "Пожалуйста, укажите страну."},
	KeyAskCity:            {en: "Which <b>city</b>?", ru: "В каком <b>городе</b>?"},
	KeyCityEmpty:          {en: "Please tell me the city.", ru: "Пожалуйста, укажите город."},
	KeyAskAddress:         {en: "What's the <b>address</b>? Send <code>-</code> to skip.", ru: "Укажите <b>адрес</b> или отправьте <code>-</code>, чтобы пропустить."},
	KeyAskTime:            {en: "When is it? Send the <b>time</b> as <code>%s</code> (your timezone: %s).", ru: "Когда? Отправьте <b>время</b> в формате <code>%s</code> (ваш часовой пояс: %s)."},
	KeyTimeBad:            {en: "I couldn't read that time. Please use <code>%s</code>.", ru: "Не удалось распознать время. Используйте формат <code>%s</code>."},
	KeyAskLink:            {en: "Got it. Paste a <b>Telegram chat link</b> for the event, or send <code>-</code> to skip.", ru: "Готово. Вставьте <b>ссылку на чат Telegram</b> или отправьте <code>-</code>, чтобы пропустить."},
	KeyAskVisibility:      {en: "Last step: who can join?", ru: "Последний шаг: кто может присоединяться?"},
	KeyBtnPublic:          {en: "🌍 Public", ru: "🌍 Публичное"},
	KeyBtnPrivateApproval: {en: "🔒 Private (approval)", ru: "🔒 Приватное (одобрение)"},
	KeyPickButtons:        {en: "Please pick an option using the buttons above.", ru: "Пожалуйста, выберите вариант с помощью кнопок выше."},
	KeyWizardExpired:      {en: "This wizard has expired. Start again with /create.", ru: "Мастер устарел. Начните заново: /create."},
	KeyUnknownOption:      {en: "Unknown option.", ru: "Неизвестный вариант."},
	KeyErrSaveLobby:       {en: "Sorry, I couldn't save the lobby. Please try again.", ru: "Не удалось сохранить лобби. Попробуйте ещё раз."},
	KeyLobbyCreated:       {en: "✅ Lobby created!", ru: "✅ Лобби создано!"},
	KeyCreatedPublicHint:  {en: "It's now listed in /lobbies for everyone to join.", ru: "Теперь оно доступно всем в /lobbies."},
	KeyCreatedPrivateHint: {en: "It's private — join requests will appear under /mylobbies for you to approve.", ru: "Оно приватное — заявки появятся в /mylobbies для одобрения."},

	KeyAlreadyIn:        {en: "You're already in this lobby ✅", ru: "Вы уже в этом лобби ✅"},
	KeyStillPending:     {en: "Your request is still pending ⏳", ru: "Ваша заявка ещё на рассмотрении ⏳"},
	KeyWasDeclined:      {en: "Your request for this lobby was declined.", ru: "Ваша заявка в это лобби была отклонена."},
	KeyBanned:           {en: "🔨 You are banned from this lobby.", ru: "🔨 Вы забанены в этом лобби."},
	KeyJoinedToast:      {en: "You're in! 🎉", ru: "Вы в деле! 🎉"},
	KeyJoinedMsg:        {en: "✅ You joined <b>%s</b>.", ru: "✅ Вы присоединились к <b>%s</b>."},
	KeyChatLine:         {en: "Chat: %s", ru: "Чат: %s"},
	KeyRequestSentToast: {en: "Request sent — waiting for approval ⏳", ru: "Заявка отправлена — ожидайте одобрения ⏳"},
	KeyRequestSentMsg:   {en: "⏳ Your request to join <b>%s</b> was sent to the admin.", ru: "⏳ Ваша заявка на участие в <b>%s</b> отправлена администратору."},
	KeyAdminNewRequest:  {en: "🔔 <b>%s</b> wants to join <b>%s</b>.", ru: "🔔 <b>%s</b> хочет присоединиться к <b>%s</b>."},
	KeyBtnApprove:       {en: "✅ Approve", ru: "✅ Одобрить"},
	KeyBtnReject:        {en: "❌ Reject", ru: "❌ Отклонить"},
	KeyLobbyGone:        {en: "That lobby no longer exists.", ru: "Этого лобби больше не существует."},
	KeyAdminOnly:        {en: "Only the lobby admin can do that.", ru: "Это может сделать только администратор лобби."},
	KeyApprovedToast:    {en: "Approved ✅", ru: "Одобрено ✅"},
	KeyApprovedEdit:     {en: "✅ Approved request for <b>%s</b>.", ru: "✅ Заявка одобрена: <b>%s</b>."},
	KeyApprovedMsg:      {en: "🎉 You were approved for <b>%s</b>!", ru: "🎉 Вас одобрили в <b>%s</b>!"},
	KeyRejectedToast:    {en: "Rejected ❌", ru: "Отклонено ❌"},
	KeyRejectedEdit:     {en: "❌ Rejected request for <b>%s</b>.", ru: "❌ Заявка отклонена: <b>%s</b>."},
	KeyRejectedMsg:      {en: "Sorry — your request to join <b>%s</b> was declined.", ru: "К сожалению, ваша заявка в <b>%s</b> отклонена."},
	KeyMalformed:        {en: "Malformed request.", ru: "Некорректный запрос."},
	KeyPendingRequest:   {en: "Pending request for <b>%s</b>:\n%s", ru: "Заявка в <b>%s</b>:\n%s"},

	KeyNoMembersYet:   {en: "No one has joined <b>%s</b> yet.", ru: "В <b>%s</b> пока никто не вступил."},
	KeyBtnRemove:      {en: "🚫 Remove", ru: "🚫 Удалить"},
	KeyBtnBan:         {en: "🔨 Ban", ru: "🔨 Забанить"},
	KeyErrLoadMembers: {en: "Sorry, could not load the members.", ru: "Не удалось загрузить участников."},
	KeyCantRemoveSelf: {en: "You can't remove yourself.", ru: "Вы не можете удалить себя."},
	KeyRemovedToast:   {en: "Removed 🚫", ru: "Удалено 🚫"},
	KeyRemovedEdit:    {en: "🚫 Removed from <b>%s</b>.", ru: "🚫 Удалён из <b>%s</b>."},
	KeyRemovedMsg:     {en: "You were removed from <b>%s</b> by the admin.", ru: "Администратор удалил вас из <b>%s</b>."},
	KeyCantBanSelf:    {en: "You can't ban yourself.", ru: "Вы не можете забанить себя."},
	KeyBannedToast:    {en: "Banned 🔨", ru: "Забанено 🔨"},
	KeyBannedEdit:     {en: "🔨 Banned from <b>%s</b>.", ru: "🔨 Забанен в <b>%s</b>."},
	KeyBannedMsg:      {en: "🔨 You were banned from <b>%s</b> and can no longer join it.", ru: "🔨 Вас забанили в <b>%s</b>, вы больше не сможете присоединиться."},

	KeyEditMenu:           {en: "What would you like to change in <b>%s</b>?", ru: "Что изменить в <b>%s</b>?"},
	KeyBtnEditName:        {en: "✏️ Name", ru: "✏️ Название"},
	KeyBtnEditDescription: {en: "📝 Description", ru: "📝 Описание"},
	KeyBtnEditCountry:     {en: "🌎 Country", ru: "🌎 Страна"},
	KeyBtnEditCity:        {en: "🏙 City", ru: "🏙 Город"},
	KeyBtnEditAddress:     {en: "🏠 Address", ru: "🏠 Адрес"},
	KeyBtnEditTime:        {en: "🕒 Time", ru: "🕒 Время"},
	KeyBtnEditLink:        {en: "💬 Chat link", ru: "💬 Ссылка на чат"},
	KeyBtnMakePublic:      {en: "🌍 Make public", ru: "🌍 Сделать публичным"},
	KeyBtnMakePrivate:     {en: "🔒 Make private", ru: "🔒 Сделать приватным"},
	KeyEditAskName:        {en: "Send the new <b>name</b>.", ru: "Отправьте новое <b>название</b>."},
	KeyEditAskDescription: {en: "Send the new <b>description</b>, or <code>-</code> to remove it.", ru: "Отправьте новое <b>описание</b> или <code>-</code>, чтобы удалить."},
	KeyEditAskCountry:     {en: "Send the new <b>country</b>.", ru: "Отправьте новую <b>страну</b>."},
	KeyEditAskCity:        {en: "Send the new <b>city</b>.", ru: "Отправьте новый <b>город</b>."},
	KeyEditAskAddress:     {en: "Send the new <b>address</b>, or <code>-</code> to remove it.", ru: "Отправьте новый <b>адрес</b> или <code>-</code>, чтобы удалить."},
	KeyEditAskTime:        {en: "Send the new <b>time</b> as <code>%s</code> (your timezone: %s).", ru: "Отправьте новое <b>время</b> в формате <code>%s</code> (ваш часовой пояс: %s)."},
	KeyEditAskLink:        {en: "Send the new <b>chat link</b>, or <code>-</code> to remove it.", ru: "Отправьте новую <b>ссылку на чат</b> или <code>-</code>, чтобы удалить."},
	KeyEditSaved:          {en: "✅ Updated <b>%s</b>.", ru: "✅ Лобби <b>%s</b> обновлено."},
	KeyEditNotify:         {en: "🔔 The lobby <b>%s</b> you joined was updated:", ru: "🔔 Лобби <b>%s</b>, к которому вы присоединились, обновлено:"},
	KeyEditExpired:        {en: "This edit session expired. Open /mylobbies again.", ru: "Сессия редактирования истекла. Откройте /mylobbies снова."},

	KeySettingsMenu:     {en: "⚙️ <b>Settings</b>\n\n🌐 Language: %s\n🕒 Timezone: %s\n🏙 City: %s\n📅 Time filter: %s\n\nWhat would you like to change?", ru: "⚙️ <b>Настройки</b>\n\n🌐 Язык: %s\n🕒 Часовой пояс: %s\n🏙 Город: %s\n📅 Фильтр по времени: %s\n\nЧто изменить?"},
	KeyBtnSetLang:       {en: "🌐 Language", ru: "🌐 Язык"},
	KeyBtnSetTimezone:   {en: "🕒 Timezone", ru: "🕒 Часовой пояс"},
	KeyBtnSetCity:       {en: "🏙 City", ru: "🏙 Город"},
	KeyBtnSetTimeFilter: {en: "📅 Time filter", ru: "📅 Фильтр по времени"},
	KeySettingsPrompt:   {en: "🌐 Choose your language:", ru: "🌐 Выберите язык:"},
	KeyBtnEnglish:       {en: "🇬🇧 English", ru: "🇬🇧 English"},
	KeyBtnRussian:       {en: "🇷🇺 Русский", ru: "🇷🇺 Русский"},
	KeyLocaleSet:        {en: "✅ Language set to English.", ru: "✅ Язык изменён на русский."},
	KeyAskTimezone:      {en: "🕒 Send your timezone as an IANA name, e.g. <code>Europe/Kyiv</code>, <code>America/New_York</code>, or <code>UTC</code>.", ru: "🕒 Отправьте часовой пояс в формате IANA, например <code>Europe/Kyiv</code>, <code>America/New_York</code> или <code>UTC</code>."},
	KeyBadTimezone:      {en: "I don't recognize that timezone. Use an IANA name like <code>Europe/Kyiv</code>.", ru: "Не удалось распознать часовой пояс. Используйте название IANA, например <code>Europe/Kyiv</code>."},
	KeyTimezoneSet:      {en: "✅ Timezone set to <b>%s</b>.", ru: "✅ Часовой пояс: <b>%s</b>."},
	KeyAskCitySetting:   {en: "🏙 Send your city to see nearby lobbies, or <code>-</code> to clear it.", ru: "🏙 Отправьте свой город, чтобы видеть ближайшие лобби, или <code>-</code>, чтобы сбросить."},
	KeyCitySet:          {en: "✅ City set to <b>%s</b>.", ru: "✅ Город: <b>%s</b>."},
	KeyCityCleared:      {en: "✅ City cleared — you'll see lobbies everywhere.", ru: "✅ Город сброшен — вы увидите лобби отовсюду."},
	KeyTimeFilterPrompt: {en: "📅 Show lobbies happening within:", ru: "📅 Показывать лобби в пределах:"},
	KeyBtnFilterDay:     {en: "Next day", ru: "24 часов"},
	KeyBtnFilterWeek:    {en: "Next week", ru: "недели"},
	KeyBtnFilterMonth:   {en: "Next month", ru: "месяца"},
	KeyBtnFilterAll:     {en: "Any time", ru: "любое время"},
	KeyTimeFilterSet:    {en: "✅ Time filter updated.", ru: "✅ Фильтр по времени обновлён."},
	KeyNotSet:           {en: "not set", ru: "не задано"},

	KeyBtnPrev:           {en: "⬅️ Prev", ru: "⬅️ Назад"},
	KeyBtnNext:           {en: "➡️ Next", ru: "➡️ Далее"},
	KeyPageLabel:         {en: "Page %d", ru: "Страница %d"},
	KeyLobbiesFilterNote: {en: "🔎 Filters — city: %s, when: %s", ru: "🔎 Фильтры — город: %s, когда: %s"},

	KeyLobbyPublic:  {en: "🌍 Public", ru: "🌍 Публичное"},
	KeyLobbyPrivate: {en: "🔒 Private — approval required", ru: "🔒 Приватное — требуется одобрение"},
	KeyJoinedCount:  {en: "👥 %d joined", ru: "👥 участников: %d"},

	KeyRoleAdmin:     {en: "👑 Admin", ru: "👑 Администратор"},
	KeyRoleMember:    {en: "🙋 Member", ru: "🙋 Участник"},
	KeyBtnDetails:    {en: "🔍 Details", ru: "🔍 Подробнее"},
	KeyNoMyLobbies:   {en: "You're not in any lobbies yet. Browse /lobbies or /create your own.", ru: "Вы пока не состоите ни в одном лобби. Откройте /lobbies или создайте своё через /create."},
	KeyNotAMember:    {en: "You don't have access to this lobby.", ru: "У вас нет доступа к этому лобби."},
	KeyStatusJoined:  {en: "✅ You've joined this lobby.", ru: "✅ Вы уже участник этого лобби."},
	KeyStatusPending: {en: "⏳ Your request is pending approval.", ru: "⏳ Ваша заявка на рассмотрении."},

	KeyBtnDeleteLobby:   {en: "🗑 Delete lobby", ru: "🗑 Удалить лобби"},
	KeyBtnLeave:         {en: "🚪 Leave", ru: "🚪 Покинуть"},
	KeyBtnConfirmDelete: {en: "🗑 Yes, delete", ru: "🗑 Да, удалить"},
	KeyBtnCancel:        {en: "✖️ Cancel", ru: "✖️ Отмена"},
	KeyDeleteConfirm:    {en: "⚠️ Delete <b>%s</b>? This cannot be undone and removes all members.", ru: "⚠️ Удалить <b>%s</b>? Это действие необратимо и удалит всех участников."},
	KeyDeletedToast:     {en: "Deleted 🗑", ru: "Удалено 🗑"},
	KeyDeletedEdit:      {en: "🗑 Lobby <b>%s</b> deleted.", ru: "🗑 Лобби <b>%s</b> удалено."},
	KeyDeletedNotify:    {en: "🗑 The lobby <b>%s</b> you joined was deleted by the admin.", ru: "🗑 Лобби <b>%s</b>, к которому вы присоединились, удалено администратором."},
	KeyLeftToast:        {en: "You left 🚪", ru: "Вы вышли 🚪"},
	KeyLeftEdit:         {en: "🚪 You left <b>%s</b>.", ru: "🚪 Вы покинули <b>%s</b>."},

	KeyBtnInvite:  {en: "🔗 Invite link", ru: "🔗 Ссылка-приглашение"},
	KeyInviteText: {en: "🔗 Share this link to invite people to <b>%s</b>:\n%s", ru: "🔗 Поделитесь этой ссылкой, чтобы пригласить в <b>%s</b>:\n%s"},
}

// Translator renders messages in a fixed locale.
type Translator struct{ loc Locale }

// For returns a Translator for the given locale.
func For(loc Locale) *Translator { return &Translator{loc: loc} }

// Locale reports the translator's locale.
func (t *Translator) Locale() Locale { return t.loc }

// T renders message k, applying fmt formatting when args are supplied.
func (t *Translator) T(k Key, args ...any) string {
	e, ok := messages[k]
	if !ok {
		return fmt.Sprintf("!missing:%d", k)
	}
	tmpl := e.en
	if t.loc == Ru {
		tmpl = e.ru
	}
	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}
