// main.go
//
// Unified Broker API - Production-ready реализация для Тинькофф Инвестиций
// Версия: 1.0.0
// Лицензия: MIT
//
// Библиотека предоставляет единый интерфейс для работы с брокерскими API,
// начиная с Тинькофф Инвестиций. Архитектура позволяет легко добавлять
// новых брокеров без изменения клиентского кода.
//
// @package main
// @author Unified Broker API Team
// @since 2025

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	investapi "github.com/tinkoff/invest-api-go-sdk/proto"
)

// ============================================================================
// Пакетные ошибки
// ============================================================================

var (
	// ErrNotImplemented возвращается, когда метод не реализован для данного брокера
	ErrNotImplemented = errors.New("метод не реализован для данного брокера")

	// ErrNotConnected возвращается, когда брокер не подключен
	ErrNotConnected = errors.New("брокер не подключен")

	// ErrInstrumentNotFound возвращается, когда инструмент не найден
	ErrInstrumentNotFound = errors.New("инструмент не найден")

	// ErrAccountNotFound возвращается, когда счет не найден
	ErrAccountNotFound = errors.New("счет не найден")

	// ErrInvalidArgument возвращается при неверных аргументах
	ErrInvalidArgument = errors.New("неверный аргумент")

	// ErrRateLimitExceeded возвращается при превышении лимита запросов
	ErrRateLimitExceeded = errors.New("превышен лимит запросов")
)

// ============================================================================
// Типы данных (Unified Model)
// ============================================================================

// InstrumentType представляет тип финансового инструмента
type InstrumentType string

const (
	InstrumentTypeStock    InstrumentType = "stock"    // Акция
	InstrumentTypeFuture   InstrumentType = "future"   // Фьючерс
	InstrumentTypeOption   InstrumentType = "option"   // Опцион
	InstrumentTypeBond     InstrumentType = "bond"     // Облигация
	InstrumentTypeETF      InstrumentType = "etf"      // ETF
	InstrumentTypeCurrency InstrumentType = "currency" // Валюта
	InstrumentTypeIndex    InstrumentType = "index"    // Индекс
	InstrumentTypeSP       InstrumentType = "sp"       // Структурная нота
)

// TradingStatus представляет статус торговли инструментом
type TradingStatus string

const (
	TradingStatusActive          TradingStatus = "active"           // Активен
	TradingStatusNormalTrading   TradingStatus = "normal_trading"   // Обычная торговля
	TradingStatusOpeningAuction  TradingStatus = "opening_auction"  // Аукцион открытия
	TradingStatusClosingAuction  TradingStatus = "closing_auction"  // Аукцион закрытия
	TradingStatusDiscreteAuction TradingStatus = "discrete_auction" // Дискретный аукцион
	TradingStatusNotAvailable    TradingStatus = "not_available"    // Недоступен для торгов
	TradingStatusSuspended       TradingStatus = "suspended"        // Приостановлен
	TradingStatusDelisted        TradingStatus = "delisted"         // Делистинг
	TradingStatusUnknown         TradingStatus = "unknown"          // Неизвестен
)

// OrderDirection представляет направление ордера
type OrderDirection string

const (
	OrderDirectionBuy  OrderDirection = "buy"  // Покупка
	OrderDirectionSell OrderDirection = "sell" // Продажа
)

// OrderType представляет тип ордера
type OrderType string

const (
	OrderTypeMarket OrderType = "market" // Рыночный
	OrderTypeLimit  OrderType = "limit"  // Лимитный
	OrderTypeBest   OrderType = "best"   // Лучшая цена
)

// OrderStatus представляет статус ордера
type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "new"       // Новый
	OrderStatusPending   OrderStatus = "pending"   // В обработке
	OrderStatusFilled    OrderStatus = "filled"    // Исполнен полностью
	OrderStatusPartially OrderStatus = "partially" // Частично исполнен
	OrderStatusCancelled OrderStatus = "cancelled" // Отменен
	OrderStatusRejected  OrderStatus = "rejected"  // Отклонен
	OrderStatusExpired   OrderStatus = "expired"   // Истек
	OrderStatusUnknown   OrderStatus = "unknown"   // Неизвестен
)

// AccountType представляет тип счета
type AccountType string

const (
	AccountTypeBroker    AccountType = "broker"    // Брокерский счет
	AccountTypeIIS       AccountType = "iis"       // ИИС
	AccountTypeInvestBox AccountType = "investbox" // Инвесткопилка
)

// AccountStatus представляет статус счета
type AccountStatus string

const (
	AccountStatusNew    AccountStatus = "new"    // Новый
	AccountStatusOpen   AccountStatus = "open"   // Открыт
	AccountStatusClosed AccountStatus = "closed" // Закрыт
)

// AccessLevel представляет уровень доступа
type AccessLevel string

const (
	AccessLevelFull     AccessLevel = "full"     // Полный
	AccessLevelReadOnly AccessLevel = "readonly" // Только чтение
	AccessLevelNoAccess AccessLevel = "noaccess" // Нет доступа
)

// CandleInterval представляет интервал свечи
type CandleInterval string

const (
	CandleInterval1Min   CandleInterval = "1min"  // 1 минута
	CandleInterval2Min   CandleInterval = "2min"  // 2 минуты
	CandleInterval3Min   CandleInterval = "3min"  // 3 минуты
	CandleInterval5Min   CandleInterval = "5min"  // 5 минут
	CandleInterval10Min  CandleInterval = "10min" // 10 минут
	CandleInterval15Min  CandleInterval = "15min" // 15 минут
	CandleInterval30Min  CandleInterval = "30min" // 30 минут
	CandleInterval1Hour  CandleInterval = "1h"    // 1 час
	CandleInterval2Hour  CandleInterval = "2h"    // 2 часа
	CandleInterval4Hour  CandleInterval = "4h"    // 4 часа
	CandleInterval1Day   CandleInterval = "1d"    // 1 день
	CandleInterval1Week  CandleInterval = "1w"    // 1 неделя
	CandleInterval1Month CandleInterval = "1M"    // 1 месяц
)

// MoneyAmount представляет денежную сумму
type MoneyAmount struct {
	Currency string  `json:"currency"` // Валюта
	Units    int64   `json:"units"`    // Целая часть
	Nano     int32   `json:"nano"`     // Дробная часть (нано)
	Value    float64 `json:"value"`    // Значение в десятичном виде
}

// NewMoneyAmount создает MoneyAmount из units и nano
func NewMoneyAmount(currency string, units int64, nano int32) MoneyAmount {
	return MoneyAmount{
		Currency: currency,
		Units:    units,
		Nano:     nano,
		Value:    float64(units) + float64(nano)/1e9,
	}
}

// NewMoneyAmountFromFloat создает MoneyAmount из float64
func NewMoneyAmountFromFloat(currency string, value float64) MoneyAmount {
	units := int64(value)
	nano := int32((value - float64(units)) * 1e9)
	return MoneyAmount{
		Currency: currency,
		Units:    units,
		Nano:     nano,
		Value:    value,
	}
}

// IsZero проверяет, равна ли сумма нулю
func (m MoneyAmount) IsZero() bool {
	return m.Units == 0 && m.Nano == 0
}

// Float64 возвращает значение как float64
func (m MoneyAmount) Float64() float64 {
	return m.Value
}

// String возвращает строковое представление
func (m MoneyAmount) String() string {
	return fmt.Sprintf("%.2f %s", m.Value, m.Currency)
}

// Instrument представляет информацию об инструменте
type Instrument struct {
	// Идентификация
	ID        string `json:"id"`                   // Внутренний ID
	BrokerID  string `json:"broker_id"`            // ID в системе брокера
	Broker    string `json:"broker"`               // Название брокера
	Ticker    string `json:"ticker"`               // Тикер
	FIGI      string `json:"figi,omitempty"`       // FIGI
	ISIN      string `json:"isin,omitempty"`       // ISIN
	ClassCode string `json:"class_code,omitempty"` // Класс-код

	// Основная информация
	Name     string         `json:"name"`     // Название
	Type     InstrumentType `json:"type"`     // Тип инструмента
	Currency string         `json:"currency"` // Валюта расчетов
	Exchange string         `json:"exchange"` // Биржа

	// Торговые параметры
	Lot               int     `json:"lot"`                 // Лотность
	MinPriceIncrement float64 `json:"min_price_increment"` // Минимальный шаг цены

	// Статусы
	TradingStatus     TradingStatus `json:"trading_status"`      // Статус торговли
	ForIIS            bool          `json:"for_iis"`             // Доступен для ИИС
	ForQual           bool          `json:"for_qual"`            // Только для квалифицированных
	ShortEnabled      bool          `json:"short_enabled"`       // Разрешен шорт
	APITradeAvailable bool          `json:"api_trade_available"` // Доступна торговля через API

	// Дополнительные поля
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Account представляет информацию о счете
type Account struct {
	ID          string        `json:"id"`                  // Внутренний ID
	BrokerID    string        `json:"broker_id"`           // ID в системе брокера
	Broker      string        `json:"broker"`              // Название брокера
	Name        string        `json:"name"`                // Название счета
	Type        AccountType   `json:"type"`                // Тип счета
	Status      AccountStatus `json:"status"`              // Статус
	OpenedAt    time.Time     `json:"opened_at"`           // Дата открытия
	ClosedAt    *time.Time    `json:"closed_at,omitempty"` // Дата закрытия
	AccessLevel AccessLevel   `json:"access_level"`        // Уровень доступа
}

// Position представляет позицию по инструменту
type Position struct {
	InstrumentID string         `json:"instrument_id"`    // ID инструмента
	Ticker       string         `json:"ticker,omitempty"` // Тикер
	Name         string         `json:"name,omitempty"`   // Название
	Type         InstrumentType `json:"type"`             // Тип инструмента

	// Количество
	Quantity     float64 `json:"quantity"`      // Количество в штуках
	QuantityLots int64   `json:"quantity_lots"` // Количество в лотах
	Blocked      float64 `json:"blocked"`       // Заблокировано

	// Цены
	AveragePrice MoneyAmount `json:"average_price"` // Средняя цена
	CurrentPrice MoneyAmount `json:"current_price"` // Текущая цена
	CurrentValue MoneyAmount `json:"current_value"` // Текущая стоимость

	// Доходность
	ProfitLoss        float64 `json:"profit_loss"`         // Абсолютная доходность
	ProfitLossPercent float64 `json:"profit_loss_percent"` // Относительная доходность
}

// Portfolio представляет портфель
type Portfolio struct {
	AccountID          string       `json:"account_id"`               // ID счета
	TotalValue         MoneyAmount  `json:"total_value"`              // Общая стоимость
	TotalValueShares   MoneyAmount  `json:"total_value_shares"`       // Акции
	TotalValueBonds    MoneyAmount  `json:"total_value_bonds"`        // Облигации
	TotalValueETF      MoneyAmount  `json:"total_value_etf"`          // ETF
	TotalValueFutures  MoneyAmount  `json:"total_value_futures"`      // Фьючерсы
	TotalValueOptions  MoneyAmount  `json:"total_value_options"`      // Опционы
	TotalValueCurrency MoneyAmount  `json:"total_value_currency"`     // Валюта
	ExpectedYield      *MoneyAmount `json:"expected_yield,omitempty"` // Ожидаемая доходность
	Positions          []Position   `json:"positions"`                // Позиции
	UpdatedAt          time.Time    `json:"updated_at"`               // Время обновления
}

// Order представляет ордер
type Order struct {
	ID            string `json:"id"`              // Внутренний ID
	BrokerOrderID string `json:"broker_order_id"` // ID в системе брокера
	AccountID     string `json:"account_id"`      // ID счета
	InstrumentID  string `json:"instrument_id"`   // ID инструмента

	Direction OrderDirection `json:"direction"`  // Направление
	OrderType OrderType      `json:"order_type"` // Тип ордера
	Status    OrderStatus    `json:"status"`     // Статус

	// Количество
	Quantity       int64 `json:"quantity"`        // Запрошено лотов
	QuantityFilled int64 `json:"quantity_filled"` // Исполнено лотов

	// Цены
	Price        float64 `json:"price,omitempty"`          // Цена ордера
	StopPrice    float64 `json:"stop_price,omitempty"`     // Стоп-цена
	AvgFillPrice float64 `json:"avg_fill_price,omitempty"` // Средняя цена исполнения

	// Временные метки
	CreatedAt time.Time `json:"created_at"` // Время создания
	UpdatedAt time.Time `json:"updated_at"` // Время обновления

	// Комиссия
	Commission float64 `json:"commission,omitempty"` // Комиссия

	// Сообщения
	Message string `json:"message,omitempty"` // Сообщение от брокера
}

// OrderResult представляет результат размещения ордера
type OrderResult struct {
	OrderID        string      `json:"order_id"`                 // Внутренний ID
	BrokerOrderID  string      `json:"broker_order_id"`          // ID в системе брокера
	Status         OrderStatus `json:"status"`                   // Статус
	QuantityFilled int64       `json:"quantity_filled"`          // Исполнено лотов
	AvgFillPrice   float64     `json:"avg_fill_price,omitempty"` // Средняя цена исполнения
	Commission     float64     `json:"commission,omitempty"`     // Комиссия
	Message        string      `json:"message,omitempty"`        // Сообщение
}

// OrderBookLevel представляет уровень стакана
type OrderBookLevel struct {
	Price    float64 `json:"price"`    // Цена
	Quantity int64   `json:"quantity"` // Количество в лотах
}

// OrderBook представляет стакан заявок
type OrderBook struct {
	InstrumentID string `json:"instrument_id"` // ID инструмента
	Depth        int    `json:"depth"`         // Глубина стакана

	Bids []OrderBookLevel `json:"bids"` // Заявки на покупку
	Asks []OrderBookLevel `json:"asks"` // Заявки на продажу

	Timestamp time.Time `json:"timestamp"` // Время формирования
	Exchange  string    `json:"exchange"`  // Биржа

	LimitUp   float64 `json:"limit_up,omitempty"`   // Верхний лимит цены
	LimitDown float64 `json:"limit_down,omitempty"` // Нижний лимит цены
}

// Candle представляет свечу
type Candle struct {
	Open  float64 `json:"open"`  // Цена открытия
	High  float64 `json:"high"`  // Максимальная цена
	Low   float64 `json:"low"`   // Минимальная цена
	Close float64 `json:"close"` // Цена закрытия

	Volume int64 `json:"volume"` // Объем в лотах

	Timestamp  time.Time `json:"timestamp"`   // Время свечи
	Interval   string    `json:"interval"`    // Интервал
	IsComplete bool      `json:"is_complete"` // Завершена ли свеча
}

// LastPrice представляет последнюю цену
type LastPrice struct {
	InstrumentID string    `json:"instrument_id"` // ID инструмента
	Price        float64   `json:"price"`         // Цена
	Timestamp    time.Time `json:"timestamp"`     // Время получения
}

// MarginInfo представляет маржинальную информацию
type MarginInfo struct {
	LiquidPortfolio      float64 `json:"liquid_portfolio"`        // Ликвидная стоимость портфеля
	StartingMargin       float64 `json:"starting_margin"`         // Начальная маржа
	MinimalMargin        float64 `json:"minimal_margin"`          // Минимальная маржа
	FundsSufficiency     float64 `json:"funds_sufficiency"`       // Уровень достаточности средств
	AmountOfMissingFunds float64 `json:"amount_of_missing_funds"` // Объем недостающих средств
	Currency             string  `json:"currency"`                // Валюта
}

// ExchangeStatus представляет состояние биржи
type ExchangeStatus struct {
	Exchange  string     `json:"exchange"`             // Биржа
	IsOpen    bool       `json:"is_open"`              // Открыта ли сейчас
	Session   string     `json:"session"`              // Текущая сессия
	NextOpen  *time.Time `json:"next_open,omitempty"`  // Следующее открытие
	NextClose *time.Time `json:"next_close,omitempty"` // Следующее закрытие
	Holiday   bool       `json:"holiday"`              // Сегодня праздник?
	Timezone  string     `json:"timezone"`             // Часовой пояс
}

// TradingDay представляет торговый день
type TradingDay struct {
	Date         time.Time `json:"date"`           // Дата
	IsTradingDay bool      `json:"is_trading_day"` // Торговый ли день
	StartTime    time.Time `json:"start_time"`     // Начало основной сессии
	EndTime      time.Time `json:"end_time"`       // Конец основной сессии
}

// TradingSchedule представляет расписание торгов
type TradingSchedule struct {
	Exchange string       `json:"exchange"` // Биржа
	Days     []TradingDay `json:"days"`     // Дни
}

// BrokerConfig конфигурация брокера
type BrokerConfig struct {
	APIKey    string        `json:"api_key"`            // API ключ
	Endpoint  string        `json:"endpoint,omitempty"` // Эндпоинт API
	Sandbox   bool          `json:"sandbox"`            // Режим песочницы
	Timeout   time.Duration `json:"timeout"`            // Таймаут запросов
	Retries   int           `json:"retries"`            // Количество повторных попыток
	KeepAlive time.Duration `json:"keep_alive"`         // Keep-alive интервал
	RateLimit int           `json:"rate_limit"`         // Лимит запросов в секунду
	LogLevel  string        `json:"log_level"`          // Уровень логирования
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() BrokerConfig {
	return BrokerConfig{
		Timeout:   30 * time.Second,
		Retries:   3,
		KeepAlive: 60 * time.Second,
		RateLimit: 10,
		LogLevel:  "info",
	}
}

// ============================================================================
// Интерфейсы брокеров
// ============================================================================

// Broker - единый интерфейс для всех брокеров
type Broker interface {
	// Информация о брокере
	Name() string
	IsAvailable() bool

	// Управление подключением
	Connect(ctx context.Context) error
	Disconnect() error
	Close() error

	// Инструменты
	GetInstruments(ctx context.Context) ([]Instrument, error)
	GetInstrument(ctx context.Context, id string) (*Instrument, error)
	SearchInstruments(ctx context.Context, query string) ([]Instrument, error)

	// Счета
	GetAccounts(ctx context.Context) ([]Account, error)

	// Портфель и позиции
	GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error)
	GetPositions(ctx context.Context, accountID string) ([]Position, error)
	GetBalance(ctx context.Context, accountID string) (map[string]float64, error)

	// Рыночные данные
	GetCandles(ctx context.Context, instrumentID string, from, to time.Time, interval CandleInterval) ([]Candle, error)
	GetLastPrice(ctx context.Context, instrumentID string) (*LastPrice, error)
	GetLastPrices(ctx context.Context, instrumentIDs []string) ([]LastPrice, error)
	GetOrderBook(ctx context.Context, instrumentID string, depth int) (*OrderBook, error)

	// Торговые операции
	PlaceOrder(ctx context.Context, order Order) (*OrderResult, error)
	CancelOrder(ctx context.Context, accountID, orderID string) error
	GetOrders(ctx context.Context, accountID string) ([]Order, error)
	GetOrder(ctx context.Context, accountID, orderID string) (*Order, error)

	// Маржинальная информация
	GetMarginInfo(ctx context.Context, accountID string) (*MarginInfo, error)
}

// ============================================================================
// Rate Limiter
// ============================================================================

// RateLimiter реализует ограничение скорости запросов
type RateLimiter struct {
	tokens chan struct{}
	rate   int
	burst  int
	stopCh chan struct{}
}

// NewRateLimiter создает новый rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	if rate <= 0 {
		rate = 10
	}
	if burst <= 0 {
		burst = rate
	}

	limiter := &RateLimiter{
		tokens: make(chan struct{}, burst),
		rate:   rate,
		burst:  burst,
		stopCh: make(chan struct{}),
	}

	// Заполняем токены
	for i := 0; i < burst; i++ {
		limiter.tokens <- struct{}{}
	}

	// Запускаем пополнение токенов
	go limiter.refill()

	return limiter
}

// refill пополняет токены с заданной скоростью
func (r *RateLimiter) refill() {
	ticker := time.NewTicker(time.Second / time.Duration(r.rate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case r.tokens <- struct{}{}:
			default:
				// Канал полон
			}
		case <-r.stopCh:
			return
		}
	}
}

// Wait ожидает доступный токен
func (r *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-r.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop останавливает rate limiter
func (r *RateLimiter) Stop() {
	close(r.stopCh)
}

// ============================================================================
// Реализация для Тинькофф
// ============================================================================

// TinkoffBroker реализация брокера для Тинькофф Инвестиций
type TinkoffBroker struct {
	config      BrokerConfig
	logger      *slog.Logger
	conn        *grpc.ClientConn
	rateLimiter *RateLimiter

	// Клиенты gRPC
	usersClient       investapi.UsersServiceClient
	instrumentsClient investapi.InstrumentsServiceClient
	marketDataClient  investapi.MarketDataServiceClient
	ordersClient      investapi.OrdersServiceClient
	operationsClient  investapi.OperationsServiceClient
	sandboxClient     investapi.SandboxServiceClient

	// Состояние
	mu         sync.RWMutex
	connected  bool
	accountMap map[string]string // маппинг внутренних ID в ID Тинькофф
}

// NewTinkoffBroker создает нового Тинькофф брокера
func NewTinkoffBroker(config BrokerConfig) (*TinkoffBroker, error) {
	if config.APIKey == "" {
		return nil, errors.New("API ключ Тинькофф обязателен")
	}

	// Устанавливаем endpoint по умолчанию
	if config.Endpoint == "" {
		if config.Sandbox {
			config.Endpoint = "sandbox-invest-public-api.tinkoff.ru:443"
		} else {
			config.Endpoint = "invest-public-api.tinkoff.ru:443"
		}
	}

	logger := slog.Default().With(
		"broker", "tinkoff",
		"sandbox", config.Sandbox,
	)

	return &TinkoffBroker{
		config:      config,
		logger:      logger,
		rateLimiter: NewRateLimiter(config.RateLimit, config.RateLimit*2),
		accountMap:  make(map[string]string),
	}, nil
}

// Name возвращает название брокера
func (b *TinkoffBroker) Name() string {
	if b.config.Sandbox {
		return "Tinkoff (Sandbox)"
	}
	return "Tinkoff"
}

// IsAvailable проверяет доступность брокера
func (b *TinkoffBroker) IsAvailable() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// Connect устанавливает соединение с API Тинькофф
func (b *TinkoffBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connected {
		return nil
	}

	b.logger.Info("подключение к API Тинькофф", "endpoint", b.config.Endpoint)

	// Настройки keepalive
	keepaliveParams := keepalive.ClientParameters{
		Time:                b.config.KeepAlive,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}

	// Настройки TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Создаем gRPC соединение
	conn, err := grpc.DialContext(
		ctx,
		b.config.Endpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithUnaryInterceptor(b.grpcUnaryInterceptor),
	)
	if err != nil {
		return fmt.Errorf("ошибка подключения к gRPC: %w", err)
	}

	b.conn = conn

	// Инициализируем клиентов
	b.usersClient = investapi.NewUsersServiceClient(conn)
	b.instrumentsClient = investapi.NewInstrumentsServiceClient(conn)
	b.marketDataClient = investapi.NewMarketDataServiceClient(conn)
	b.ordersClient = investapi.NewOrdersServiceClient(conn)
	b.operationsClient = investapi.NewOperationsServiceClient(conn)

	if b.config.Sandbox {
		b.sandboxClient = investapi.NewSandboxServiceClient(conn)
	}

	b.connected = true
	b.logger.Info("успешно подключено к API Тинькофф")

	return nil
}

// Disconnect закрывает соединение
func (b *TinkoffBroker) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}

	b.logger.Info("отключение от API Тинькофф")

	if b.rateLimiter != nil {
		b.rateLimiter.Stop()
	}

	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			return fmt.Errorf("ошибка при закрытии соединения: %w", err)
		}
	}

	b.connected = false
	b.logger.Info("отключено от API Тинькофф")

	return nil
}

// Close закрывает брокера
func (b *TinkoffBroker) Close() error {
	return b.Disconnect()
}

// gRPC интерцептор для добавления метаданных и rate limiting
func (b *TinkoffBroker) grpcUnaryInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Rate limiting
	if err := b.rateLimiter.Wait(ctx); err != nil {
		return ErrRateLimitExceeded
	}

	// Добавляем токен в метаданные
	md := metadata.New(map[string]string{
		"Authorization": "Bearer " + b.config.APIKey,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Выполняем запрос с таймаутом
	ctxWithTimeout, cancel := context.WithTimeout(ctx, b.config.Timeout)
	defer cancel()

	start := time.Now()
	err := invoker(ctxWithTimeout, method, req, reply, cc, opts...)
	duration := time.Since(start)

	// Логируем запрос
	level := slog.LevelDebug
	if err != nil {
		level = slog.LevelError
	}

	b.logger.Log(ctx, level, "gRPC запрос",
		"method", method,
		"duration_ms", duration.Milliseconds(),
		"error", err,
	)

	return err
}

// ============================================================================
// Мапперы для Тинькофф
// ============================================================================

// mapInstrumentType маппинг типа инструмента из Тинькофф
func mapInstrumentType(t investapi.InstrumentType) InstrumentType {
	switch t {
	case investapi.InstrumentType_INSTRUMENT_TYPE_SHARE:
		return InstrumentTypeStock
	case investapi.InstrumentType_INSTRUMENT_TYPE_BOND:
		return InstrumentTypeBond
	case investapi.InstrumentType_INSTRUMENT_TYPE_ETF:
		return InstrumentTypeETF
	case investapi.InstrumentType_INSTRUMENT_TYPE_FUTURES:
		return InstrumentTypeFuture
	case investapi.InstrumentType_INSTRUMENT_TYPE_OPTION:
		return InstrumentTypeOption
	case investapi.InstrumentType_INSTRUMENT_TYPE_CURRENCY:
		return InstrumentTypeCurrency
	default:
		return InstrumentTypeStock
	}
}

// mapTradingStatus маппинг статуса торговли из Тинькофф
func mapTradingStatus(status investapi.SecurityTradingStatus) TradingStatus {
	switch status {
	case investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_NORMAL_TRADING:
		return TradingStatusNormalTrading
	case investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_NOT_AVAILABLE_FOR_TRADING:
		return TradingStatusNotAvailable
	case investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_OPENING_PERIOD,
		investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_OPENING_AUCTION_PERIOD:
		return TradingStatusOpeningAuction
	case investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_CLOSING_PERIOD,
		investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_CLOSING_AUCTION:
		return TradingStatusClosingAuction
	case investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_DISCRETE_AUCTION:
		return TradingStatusDiscreteAuction
	case investapi.SecurityTradingStatus_SECURITY_TRADING_STATUS_SESSION_CLOSE:
		return TradingStatusSuspended
	default:
		return TradingStatusUnknown
	}
}

// mapOrderDirection маппинг направления ордера из Тинькофф
func mapOrderDirection(dir investapi.OrderDirection) OrderDirection {
	switch dir {
	case investapi.OrderDirection_ORDER_DIRECTION_BUY:
		return OrderDirectionBuy
	case investapi.OrderDirection_ORDER_DIRECTION_SELL:
		return OrderDirectionSell
	default:
		return OrderDirectionBuy
	}
}

// mapOrderDirectionToTinkoff маппинг направления ордера в Тинькофф
func mapOrderDirectionToTinkoff(dir OrderDirection) investapi.OrderDirection {
	switch dir {
	case OrderDirectionBuy:
		return investapi.OrderDirection_ORDER_DIRECTION_BUY
	case OrderDirectionSell:
		return investapi.OrderDirection_ORDER_DIRECTION_SELL
	default:
		return investapi.OrderDirection_ORDER_DIRECTION_BUY
	}
}

// mapOrderType маппинг типа ордера из Тинькофф
func mapOrderType(t investapi.OrderType) OrderType {
	switch t {
	case investapi.OrderType_ORDER_TYPE_LIMIT:
		return OrderTypeLimit
	case investapi.OrderType_ORDER_TYPE_MARKET:
		return OrderTypeMarket
	default:
		return OrderTypeMarket
	}
}

// mapOrderTypeToTinkoff маппинг типа ордера в Тинькофф
func mapOrderTypeToTinkoff(t OrderType) investapi.OrderType {
	switch t {
	case OrderTypeLimit:
		return investapi.OrderType_ORDER_TYPE_LIMIT
	case OrderTypeMarket:
		return investapi.OrderType_ORDER_TYPE_MARKET
	default:
		return investapi.OrderType_ORDER_TYPE_MARKET
	}
}

// mapOrderStatus маппинг статуса ордера из Тинькофф
func mapOrderStatus(status investapi.OrderExecutionReportStatus) OrderStatus {
	switch status {
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_NEW:
		return OrderStatusNew
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL:
		return OrderStatusFilled
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL:
		return OrderStatusPartially
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED:
		return OrderStatusCancelled
	case investapi.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED:
		return OrderStatusRejected
	default:
		return OrderStatusUnknown
	}
}

// mapAccountType маппинг типа счета из Тинькофф
func mapAccountType(t investapi.AccountType) AccountType {
	switch t {
	case investapi.AccountType_ACCOUNT_TYPE_TINKOFF:
		return AccountTypeBroker
	case investapi.AccountType_ACCOUNT_TYPE_TINKOFF_IIS:
		return AccountTypeIIS
	case investapi.AccountType_ACCOUNT_TYPE_INVEST_BOX:
		return AccountTypeInvestBox
	default:
		return AccountTypeBroker
	}
}

// mapAccountStatus маппинг статуса счета из Тинькофф
func mapAccountStatus(status investapi.AccountStatus) AccountStatus {
	switch status {
	case investapi.AccountStatus_ACCOUNT_STATUS_NEW:
		return AccountStatusNew
	case investapi.AccountStatus_ACCOUNT_STATUS_OPEN:
		return AccountStatusOpen
	case investapi.AccountStatus_ACCOUNT_STATUS_CLOSED:
		return AccountStatusClosed
	default:
		return AccountStatusOpen
	}
}

// mapAccessLevel маппинг уровня доступа из Тинькофф
func mapAccessLevel(level investapi.AccessLevel) AccessLevel {
	switch level {
	case investapi.AccessLevel_ACCOUNT_ACCESS_LEVEL_FULL_ACCESS:
		return AccessLevelFull
	case investapi.AccessLevel_ACCOUNT_ACCESS_LEVEL_READ_ONLY:
		return AccessLevelReadOnly
	case investapi.AccessLevel_ACCOUNT_ACCESS_LEVEL_NO_ACCESS:
		return AccessLevelNoAccess
	default:
		return AccessLevelReadOnly
	}
}

// quotationToFloat преобразует Quotation в float64
func quotationToFloat(q *investapi.Quotation) float64 {
	if q == nil {
		return 0
	}
	return float64(q.Units) + float64(q.Nano)/1e9
}

// moneyValueToFloat преобразует MoneyValue в float64
func moneyValueToFloat(mv *investapi.MoneyValue) float64 {
	if mv == nil {
		return 0
	}
	return float64(mv.Units) + float64(mv.Nano)/1e9
}

// mapCandleIntervalToTinkoff маппинг интервала свечи в Тинькофф
func mapCandleIntervalToTinkoff(interval CandleInterval) (investapi.CandleInterval, error) {
	switch interval {
	case CandleInterval1Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_1_MIN, nil
	case CandleInterval2Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_2_MIN, nil
	case CandleInterval3Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_3_MIN, nil
	case CandleInterval5Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_5_MIN, nil
	case CandleInterval10Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_10_MIN, nil
	case CandleInterval15Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_15_MIN, nil
	case CandleInterval30Min:
		return investapi.CandleInterval_CANDLE_INTERVAL_30_MIN, nil
	case CandleInterval1Hour:
		return investapi.CandleInterval_CANDLE_INTERVAL_HOUR, nil
	case CandleInterval2Hour:
		return investapi.CandleInterval_CANDLE_INTERVAL_2_HOUR, nil
	case CandleInterval4Hour:
		return investapi.CandleInterval_CANDLE_INTERVAL_4_HOUR, nil
	case CandleInterval1Day:
		return investapi.CandleInterval_CANDLE_INTERVAL_DAY, nil
	case CandleInterval1Week:
		return investapi.CandleInterval_CANDLE_INTERVAL_WEEK, nil
	case CandleInterval1Month:
		return investapi.CandleInterval_CANDLE_INTERVAL_MONTH, nil
	default:
		return 0, fmt.Errorf("неподдерживаемый интервал: %s", interval)
	}
}

// ============================================================================
// Реализация методов Тинькофф
// ============================================================================

// GetAccounts получает счета пользователя
func (b *TinkoffBroker) GetAccounts(ctx context.Context) ([]Account, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	var resp *investapi.GetAccountsResponse
	var err error

	if b.config.Sandbox && b.sandboxClient != nil {
		resp, err = b.sandboxClient.GetSandboxAccounts(ctx, &investapi.GetAccountsRequest{})
	} else {
		resp, err = b.usersClient.GetAccounts(ctx, &investapi.GetAccountsRequest{})
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения счетов: %w", err)
	}

	var accounts []Account
	for i, acc := range resp.Accounts {
		internalID := fmt.Sprintf("tinkoff_%d", i)
		b.accountMap[internalID] = acc.Id

		account := Account{
			ID:          internalID,
			BrokerID:    acc.Id,
			Broker:      "tinkoff",
			Name:        acc.Name,
			Type:        mapAccountType(acc.Type),
			Status:      mapAccountStatus(acc.Status),
			OpenedAt:    acc.OpenedDate.AsTime(),
			AccessLevel: mapAccessLevel(acc.AccessLevel),
		}

		if acc.ClosedDate != nil {
			t := acc.ClosedDate.AsTime()
			account.ClosedAt = &t
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// GetInstruments получает все инструменты
func (b *TinkoffBroker) GetInstruments(ctx context.Context) ([]Instrument, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	var allInstruments []Instrument

	// Получаем акции
	shares, err := b.getShares(ctx)
	if err != nil {
		b.logger.Error("ошибка получения акций", "error", err)
	} else {
		allInstruments = append(allInstruments, shares...)
	}

	// Получаем облигации
	bonds, err := b.getBonds(ctx)
	if err != nil {
		b.logger.Error("ошибка получения облигаций", "error", err)
	} else {
		allInstruments = append(allInstruments, bonds...)
	}

	// Получаем ETF
	etfs, err := b.getEtfs(ctx)
	if err != nil {
		b.logger.Error("ошибка получения ETF", "error", err)
	} else {
		allInstruments = append(allInstruments, etfs...)
	}

	// Получаем фьючерсы
	futures, err := b.getFutures(ctx)
	if err != nil {
		b.logger.Error("ошибка получения фьючерсов", "error", err)
	} else {
		allInstruments = append(allInstruments, futures...)
	}

	// Получаем валюты
	currencies, err := b.getCurrencies(ctx)
	if err != nil {
		b.logger.Error("ошибка получения валют", "error", err)
	} else {
		allInstruments = append(allInstruments, currencies...)
	}

	b.logger.Info("получены инструменты", "count", len(allInstruments))
	return allInstruments, nil
}

// getShares получает акции
func (b *TinkoffBroker) getShares(ctx context.Context) ([]Instrument, error) {
	resp, err := b.instrumentsClient.Shares(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus: investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE,
	})
	if err != nil {
		return nil, err
	}

	var instruments []Instrument
	for _, share := range resp.Instruments {
		instruments = append(instruments, Instrument{
			ID:                "tinkoff_" + share.Uid,
			BrokerID:          share.Uid,
			Broker:            "tinkoff",
			Ticker:            share.Ticker,
			FIGI:              share.Figi,
			ISIN:              share.Isin,
			ClassCode:         share.ClassCode,
			Name:              share.Name,
			Type:              InstrumentTypeStock,
			Currency:          share.Currency,
			Exchange:          share.Exchange,
			Lot:               int(share.Lot),
			MinPriceIncrement: quotationToFloat(share.MinPriceIncrement),
			TradingStatus:     mapTradingStatus(share.TradingStatus),
			ForIIS:            share.ForIisFlag,
			ForQual:           share.ForQualInvestorFlag,
			ShortEnabled:      share.ShortEnabledFlag,
			APITradeAvailable: share.ApiTradeAvailableFlag,
		})
	}

	return instruments, nil
}

// getBonds получает облигации
func (b *TinkoffBroker) getBonds(ctx context.Context) ([]Instrument, error) {
	resp, err := b.instrumentsClient.Bonds(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus: investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE,
	})
	if err != nil {
		return nil, err
	}

	var instruments []Instrument
	for _, bond := range resp.Instruments {
		instruments = append(instruments, Instrument{
			ID:                "tinkoff_" + bond.Uid,
			BrokerID:          bond.Uid,
			Broker:            "tinkoff",
			Ticker:            bond.Ticker,
			FIGI:              bond.Figi,
			ISIN:              bond.Isin,
			ClassCode:         bond.ClassCode,
			Name:              bond.Name,
			Type:              InstrumentTypeBond,
			Currency:          bond.Currency,
			Exchange:          bond.Exchange,
			Lot:               int(bond.Lot),
			MinPriceIncrement: quotationToFloat(bond.MinPriceIncrement),
			TradingStatus:     mapTradingStatus(bond.TradingStatus),
			ForIIS:            bond.ForIisFlag,
			ForQual:           bond.ForQualInvestorFlag,
			ShortEnabled:      bond.ShortEnabledFlag,
			APITradeAvailable: bond.ApiTradeAvailableFlag,
		})
	}

	return instruments, nil
}

// getEtfs получает ETF
func (b *TinkoffBroker) getEtfs(ctx context.Context) ([]Instrument, error) {
	resp, err := b.instrumentsClient.Etfs(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus: investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE,
	})
	if err != nil {
		return nil, err
	}

	var instruments []Instrument
	for _, etf := range resp.Instruments {
		instruments = append(instruments, Instrument{
			ID:                "tinkoff_" + etf.Uid,
			BrokerID:          etf.Uid,
			Broker:            "tinkoff",
			Ticker:            etf.Ticker,
			FIGI:              etf.Figi,
			ISIN:              etf.Isin,
			ClassCode:         etf.ClassCode,
			Name:              etf.Name,
			Type:              InstrumentTypeETF,
			Currency:          etf.Currency,
			Exchange:          etf.Exchange,
			Lot:               int(etf.Lot),
			MinPriceIncrement: quotationToFloat(etf.MinPriceIncrement),
			TradingStatus:     mapTradingStatus(etf.TradingStatus),
			ForIIS:            etf.ForIisFlag,
			ForQual:           etf.ForQualInvestorFlag,
			ShortEnabled:      etf.ShortEnabledFlag,
			APITradeAvailable: etf.ApiTradeAvailableFlag,
		})
	}

	return instruments, nil
}

// getFutures получает фьючерсы
func (b *TinkoffBroker) getFutures(ctx context.Context) ([]Instrument, error) {
	resp, err := b.instrumentsClient.Futures(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus: investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE,
	})
	if err != nil {
		return nil, err
	}

	var instruments []Instrument
	for _, future := range resp.Instruments {
		instruments = append(instruments, Instrument{
			ID:                "tinkoff_" + future.Uid,
			BrokerID:          future.Uid,
			Broker:            "tinkoff",
			Ticker:            future.Ticker,
			FIGI:              future.Figi,
			ClassCode:         future.ClassCode,
			Name:              future.Name,
			Type:              InstrumentTypeFuture,
			Currency:          future.Currency,
			Exchange:          future.Exchange,
			Lot:               int(future.Lot),
			MinPriceIncrement: quotationToFloat(future.MinPriceIncrement),
			TradingStatus:     mapTradingStatus(future.TradingStatus),
			ForIIS:            future.ForIisFlag,
			ForQual:           future.ForQualInvestorFlag,
			ShortEnabled:      future.ShortEnabledFlag,
			APITradeAvailable: future.ApiTradeAvailableFlag,
		})
	}

	return instruments, nil
}

// getCurrencies получает валюты
func (b *TinkoffBroker) getCurrencies(ctx context.Context) ([]Instrument, error) {
	resp, err := b.instrumentsClient.Currencies(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus: investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE,
	})
	if err != nil {
		return nil, err
	}

	var instruments []Instrument
	for _, currency := range resp.Instruments {
		instruments = append(instruments, Instrument{
			ID:                "tinkoff_" + currency.Uid,
			BrokerID:          currency.Uid,
			Broker:            "tinkoff",
			Ticker:            currency.Ticker,
			FIGI:              currency.Figi,
			ISIN:              currency.Isin,
			ClassCode:         currency.ClassCode,
			Name:              currency.Name,
			Type:              InstrumentTypeCurrency,
			Currency:          currency.Currency,
			Exchange:          currency.Exchange,
			Lot:               int(currency.Lot),
			MinPriceIncrement: quotationToFloat(currency.MinPriceIncrement),
			TradingStatus:     mapTradingStatus(currency.TradingStatus),
			ForIIS:            currency.ForIisFlag,
			ForQual:           currency.ForQualInvestorFlag,
			ShortEnabled:      currency.ShortEnabledFlag,
			APITradeAvailable: currency.ApiTradeAvailableFlag,
		})
	}

	return instruments, nil
}

// GetInstrument получает инструмент по ID
func (b *TinkoffBroker) GetInstrument(ctx context.Context, id string) (*Instrument, error) {
	// Пробуем получить по UID
	resp, err := b.instrumentsClient.GetInstrumentBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID,
		Id:     strings.TrimPrefix(id, "tinkoff_"),
	})
	if err == nil && resp != nil && resp.Instrument != nil {
		return b.mapAnyInstrument(resp.Instrument), nil
	}

	// Пробуем получить по FIGI
	resp, err = b.instrumentsClient.GetInstrumentBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_FIGI,
		Id:     id,
	})
	if err == nil && resp != nil && resp.Instrument != nil {
		return b.mapAnyInstrument(resp.Instrument), nil
	}

	return nil, ErrInstrumentNotFound
}

// mapAnyInstrument маппинг любого инструмента
func (b *TinkoffBroker) mapAnyInstrument(instr *investapi.Instrument) *Instrument {
	if instr == nil {
		return nil
	}

	return &Instrument{
		ID:                "tinkoff_" + instr.Uid,
		BrokerID:          instr.Uid,
		Broker:            "tinkoff",
		Ticker:            instr.Ticker,
		FIGI:              instr.Figi,
		ISIN:              instr.Isin,
		ClassCode:         instr.ClassCode,
		Name:              instr.Name,
		Type:              mapInstrumentType(instr.InstrumentKind),
		Currency:          instr.Currency,
		Exchange:          instr.Exchange,
		Lot:               int(instr.Lot),
		MinPriceIncrement: quotationToFloat(instr.MinPriceIncrement),
		TradingStatus:     mapTradingStatus(instr.TradingStatus),
		ForIIS:            instr.ForIisFlag,
		ForQual:           instr.ForQualInvestorFlag,
		ShortEnabled:      instr.ShortEnabledFlag,
		APITradeAvailable: instr.ApiTradeAvailableFlag,
	}
}

// SearchInstruments ищет инструменты по запросу
func (b *TinkoffBroker) SearchInstruments(ctx context.Context, query string) ([]Instrument, error) {
	resp, err := b.instrumentsClient.FindInstrument(ctx, &investapi.FindInstrumentRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска инструментов: %w", err)
	}

	var instruments []Instrument
	for _, instr := range resp.Instruments {
		// Получаем полную информацию по инструменту
		fullInstr, err := b.GetInstrumentByFIGI(ctx, instr.Figi)
		if err != nil {
			continue
		}
		if fullInstr != nil {
			instruments = append(instruments, *fullInstr)
		}
	}

	return instruments, nil
}

// GetInstrumentByFIGI получает инструмент по FIGI
func (b *TinkoffBroker) GetInstrumentByFIGI(ctx context.Context, figi string) (*Instrument, error) {
	resp, err := b.instrumentsClient.GetInstrumentBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_FIGI,
		Id:     figi,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения инструмента по FIGI: %w", err)
	}

	return b.mapAnyInstrument(resp.Instrument), nil
}

// GetPortfolio получает портфель
func (b *TinkoffBroker) GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	brokerID, err := b.getBrokerAccountID(accountID)
	if err != nil {
		return nil, err
	}

	var resp *investapi.PortfolioResponse

	if b.config.Sandbox && b.sandboxClient != nil {
		resp, err = b.sandboxClient.GetSandboxPortfolio(ctx, &investapi.PortfolioRequest{
			AccountId: brokerID,
		})
	} else {
		resp, err = b.operationsClient.GetPortfolio(ctx, &investapi.PortfolioRequest{
			AccountId: brokerID,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения портфеля: %w", err)
	}

	portfolio := &Portfolio{
		AccountID:          accountID,
		TotalValue:         NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountPortfolio)),
		TotalValueShares:   NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountShares)),
		TotalValueBonds:    NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountBonds)),
		TotalValueETF:      NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountEtf)),
		TotalValueFutures:  NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountFutures)),
		TotalValueOptions:  NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountOptions)),
		TotalValueCurrency: NewMoneyAmountFromFloat("RUB", moneyValueToFloat(resp.TotalAmountCurrencies)),
		UpdatedAt:          time.Now(),
	}

	if resp.ExpectedYield != nil {
		yield := NewMoneyAmountFromFloat("RUB", quotationToFloat(resp.ExpectedYield))
		portfolio.ExpectedYield = &yield
	}

	// Маппим позиции
	for _, pos := range resp.Positions {
		position := b.mapPortfolioPosition(pos)
		portfolio.Positions = append(portfolio.Positions, position)
	}

	return portfolio, nil
}

// mapPortfolioPosition маппинг позиции портфеля
// mapPortfolioPosition маппинг позиции портфеля
func (b *TinkoffBroker) mapPortfolioPosition(pos *investapi.PortfolioPosition) Position {
	avgPrice := NewMoneyAmountFromFloat(pos.AveragePositionPrice.Currency, moneyValueToFloat(pos.AveragePositionPrice))
	currentPrice := NewMoneyAmountFromFloat(pos.CurrentPrice.Currency, moneyValueToFloat(pos.CurrentPrice))

	// Исправлено: конвертируем int64 в float64
	quantity := float64(pos.Quantity.Units) + float64(pos.Quantity.Nano)/1e9
	quantityLots := float64(pos.Quantity.Units)

	currentValue := NewMoneyAmountFromFloat(
		pos.CurrentPrice.Currency,
		moneyValueToFloat(pos.CurrentPrice)*quantityLots, // quantityLots уже float64
	)

	// Расчет доходности
	profitLoss := currentValue.Value - (avgPrice.Value * quantityLots)
	profitLossPercent := 0.0
	if avgPrice.Value > 0 {
		profitLossPercent = (currentPrice.Value - avgPrice.Value) / avgPrice.Value * 100
	}

	return Position{
		InstrumentID:      "tinkoff_" + pos.InstrumentUid,
		Ticker:            "",
		Type:              b.mapInstrumentTypeFromString(pos.InstrumentType),
		Quantity:          quantity,
		QuantityLots:      int64(pos.Quantity.Units),
		Blocked:           float64(pos.BlockedLots.Units) + float64(pos.BlockedLots.Nano)/1e9,
		AveragePrice:      avgPrice,
		CurrentPrice:      currentPrice,
		CurrentValue:      currentValue,
		ProfitLoss:        profitLoss,
		ProfitLossPercent: profitLossPercent,
	}
}

// mapInstrumentTypeFromString маппинг строкового типа в InstrumentType
func (b *TinkoffBroker) mapInstrumentTypeFromString(t string) InstrumentType {
	switch t {
	case "share":
		return InstrumentTypeStock
	case "bond":
		return InstrumentTypeBond
	case "etf":
		return InstrumentTypeETF
	case "future":
		return InstrumentTypeFuture
	case "option":
		return InstrumentTypeOption
	case "currency":
		return InstrumentTypeCurrency
	default:
		return InstrumentTypeStock
	}
}

// GetPositions получает позиции
func (b *TinkoffBroker) GetPositions(ctx context.Context, accountID string) ([]Position, error) {
	portfolio, err := b.GetPortfolio(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return portfolio.Positions, nil
}

// GetBalance получает баланс
func (b *TinkoffBroker) GetBalance(ctx context.Context, accountID string) (map[string]float64, error) {
	brokerID, err := b.getBrokerAccountID(accountID)
	if err != nil {
		return nil, err
	}

	var resp *investapi.PositionsResponse

	if b.config.Sandbox && b.sandboxClient != nil {
		resp, err = b.sandboxClient.GetSandboxPositions(ctx, &investapi.PositionsRequest{
			AccountId: brokerID,
		})
	} else {
		resp, err = b.operationsClient.GetPositions(ctx, &investapi.PositionsRequest{
			AccountId: brokerID,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения позиций: %w", err)
	}

	balances := make(map[string]float64)
	for _, money := range resp.Money {
		balances[money.Currency] = moneyValueToFloat(money)
	}

	return balances, nil
}

// GetCandles получает свечи
func (b *TinkoffBroker) GetCandles(ctx context.Context, instrumentID string, from, to time.Time, interval CandleInterval) ([]Candle, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	// Получаем инструмент, чтобы узнать FIGI
	instr, err := b.GetInstrument(ctx, instrumentID)
	if err != nil {
		return nil, err
	}

	// Маппинг интервала
	candleInterval, err := mapCandleIntervalToTinkoff(interval)
	if err != nil {
		return nil, err
	}

	// Конвертируем time.Time в timestamppb.Timestamp
	fromProto := timestamppb.New(from)
	toProto := timestamppb.New(to)

	resp, err := b.marketDataClient.GetCandles(ctx, &investapi.GetCandlesRequest{
		Figi:         instr.FIGI,
		From:         fromProto,
		To:           toProto,
		Interval:     candleInterval,
		InstrumentId: instr.FIGI,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свечей: %w", err)
	}

	var candles []Candle
	for _, c := range resp.Candles {
		candles = append(candles, Candle{
			Open:       quotationToFloat(c.Open),
			High:       quotationToFloat(c.High),
			Low:        quotationToFloat(c.Low),
			Close:      quotationToFloat(c.Close),
			Volume:     c.Volume,
			Timestamp:  c.Time.AsTime(),
			Interval:   string(interval),
			IsComplete: c.IsComplete,
		})
	}

	return candles, nil
}

// GetLastPrice получает последнюю цену
func (b *TinkoffBroker) GetLastPrice(ctx context.Context, instrumentID string) (*LastPrice, error) {
	prices, err := b.GetLastPrices(ctx, []string{instrumentID})
	if err != nil {
		return nil, err
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("цена не найдена для инструмента %s", instrumentID)
	}

	return &prices[0], nil
}

// GetLastPrices получает последние цены для нескольких инструментов
func (b *TinkoffBroker) GetLastPrices(ctx context.Context, instrumentIDs []string) ([]LastPrice, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	var figis []string
	for _, id := range instrumentIDs {
		instr, err := b.GetInstrument(ctx, id)
		if err != nil {
			continue
		}
		figis = append(figis, instr.FIGI)
	}

	if len(figis) == 0 {
		return nil, errors.New("не удалось получить FIGI для инструментов")
	}

	resp, err := b.marketDataClient.GetLastPrices(ctx, &investapi.GetLastPricesRequest{
		Figi: figis,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения последних цен: %w", err)
	}

	var prices []LastPrice
	for _, price := range resp.LastPrices {
		prices = append(prices, LastPrice{
			InstrumentID: "tinkoff_" + price.InstrumentUid,
			Price:        quotationToFloat(price.Price),
			Timestamp:    price.Time.AsTime(),
		})
	}

	return prices, nil
}

// GetOrderBook получает стакан заявок
func (b *TinkoffBroker) GetOrderBook(ctx context.Context, instrumentID string, depth int) (*OrderBook, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	// Проверяем глубину стакана
	if depth < 1 || depth > 50 {
		return nil, fmt.Errorf("%w: глубина стакана должна быть от 1 до 50", ErrInvalidArgument)
	}

	instr, err := b.GetInstrument(ctx, instrumentID)
	if err != nil {
		return nil, err
	}

	resp, err := b.marketDataClient.GetOrderBook(ctx, &investapi.GetOrderBookRequest{
		Figi:         instr.FIGI,
		Depth:        int32(depth),
		InstrumentId: instr.FIGI,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения стакана: %w", err)
	}

	// Исправлено: используем OrderbookTs вместо Time
	orderBook := &OrderBook{
		InstrumentID: instrumentID,
		Depth:        int(resp.Depth),
		Timestamp:    resp.OrderbookTs.AsTime(), // Правильное поле
		Exchange:     "MOEX",
	}

	// Маппим заявки на покупку
	for _, bid := range resp.Bids {
		orderBook.Bids = append(orderBook.Bids, OrderBookLevel{
			Price:    quotationToFloat(bid.Price),
			Quantity: bid.Quantity,
		})
	}

	// Маппим заявки на продажу
	for _, ask := range resp.Asks {
		orderBook.Asks = append(orderBook.Asks, OrderBookLevel{
			Price:    quotationToFloat(ask.Price),
			Quantity: ask.Quantity,
		})
	}

	if resp.LimitUp != nil {
		orderBook.LimitUp = quotationToFloat(resp.LimitUp)
	}

	if resp.LimitDown != nil {
		orderBook.LimitDown = quotationToFloat(resp.LimitDown)
	}

	return orderBook, nil
}

// PlaceOrder размещает ордер
func (b *TinkoffBroker) PlaceOrder(ctx context.Context, order Order) (*OrderResult, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	brokerID, err := b.getBrokerAccountID(order.AccountID)
	if err != nil {
		return nil, err
	}

	instr, err := b.GetInstrument(ctx, order.InstrumentID)
	if err != nil {
		return nil, err
	}

	// Создаем запрос
	req := &investapi.PostOrderRequest{
		Figi:         instr.FIGI,
		Quantity:     order.Quantity,
		Direction:    mapOrderDirectionToTinkoff(order.Direction),
		AccountId:    brokerID,
		OrderType:    mapOrderTypeToTinkoff(order.OrderType),
		InstrumentId: instr.FIGI,
	}

	// Устанавливаем цену для лимитных ордеров
	if order.OrderType == OrderTypeLimit && order.Price > 0 {
		priceUnits := int64(order.Price)
		priceNano := int32((order.Price - float64(priceUnits)) * 1e9)
		req.Price = &investapi.Quotation{
			Units: priceUnits,
			Nano:  priceNano,
		}
	}

	// Генерируем ID идемпотентности
	req.OrderId = fmt.Sprintf("order_%d", time.Now().UnixNano())

	var resp *investapi.PostOrderResponse

	if b.config.Sandbox && b.sandboxClient != nil {
		resp, err = b.sandboxClient.PostSandboxOrder(ctx, req)
	} else {
		resp, err = b.ordersClient.PostOrder(ctx, req)
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка размещения ордера: %w", err)
	}

	result := &OrderResult{
		OrderID:        fmt.Sprintf("tinkoff_order_%s", resp.OrderId),
		BrokerOrderID:  resp.OrderId,
		Status:         mapOrderStatus(resp.ExecutionReportStatus),
		QuantityFilled: resp.LotsExecuted,
		Message:        resp.Message,
	}

	if resp.ExecutedOrderPrice != nil {
		result.AvgFillPrice = moneyValueToFloat(resp.ExecutedOrderPrice)
	}

	if resp.ExecutedCommission != nil {
		result.Commission = moneyValueToFloat(resp.ExecutedCommission)
	}

	return result, nil
}

// CancelOrder отменяет ордер
func (b *TinkoffBroker) CancelOrder(ctx context.Context, accountID, orderID string) error {
	if !b.connected {
		return ErrNotConnected
	}

	brokerID, err := b.getBrokerAccountID(accountID)
	if err != nil {
		return err
	}

	// Убираем префикс
	brokerOrderID := strings.TrimPrefix(orderID, "tinkoff_order_")

	var resp *investapi.CancelOrderResponse

	if b.config.Sandbox && b.sandboxClient != nil {
		resp, err = b.sandboxClient.CancelSandboxOrder(ctx, &investapi.CancelOrderRequest{
			AccountId: brokerID,
			OrderId:   brokerOrderID,
		})
	} else {
		resp, err = b.ordersClient.CancelOrder(ctx, &investapi.CancelOrderRequest{
			AccountId: brokerID,
			OrderId:   brokerOrderID,
		})
	}

	if err != nil {
		return fmt.Errorf("ошибка отмены ордера: %w", err)
	}

	b.logger.Info("ордер отменен", "order_id", orderID, "time", resp.Time.AsTime())
	return nil
}

// GetOrders получает список активных ордеров
func (b *TinkoffBroker) GetOrders(ctx context.Context, accountID string) ([]Order, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	brokerID, err := b.getBrokerAccountID(accountID)
	if err != nil {
		return nil, err
	}

	var resp *investapi.GetOrdersResponse

	if b.config.Sandbox && b.sandboxClient != nil {
		resp, err = b.sandboxClient.GetSandboxOrders(ctx, &investapi.GetOrdersRequest{
			AccountId: brokerID,
		})
	} else {
		resp, err = b.ordersClient.GetOrders(ctx, &investapi.GetOrdersRequest{
			AccountId: brokerID,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения ордеров: %w", err)
	}

	var orders []Order
	for _, orderState := range resp.Orders {
		order := b.mapOrderState(orderState, accountID)
		orders = append(orders, order)
	}

	return orders, nil
}

// GetOrder получает информацию об ордере
func (b *TinkoffBroker) GetOrder(ctx context.Context, accountID, orderID string) (*Order, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	brokerID, err := b.getBrokerAccountID(accountID)
	if err != nil {
		return nil, err
	}

	brokerOrderID := strings.TrimPrefix(orderID, "tinkoff_order_")

	var orderState *investapi.OrderState

	if b.config.Sandbox && b.sandboxClient != nil {
		orderState, err = b.sandboxClient.GetSandboxOrderState(ctx, &investapi.GetOrderStateRequest{
			AccountId: brokerID,
			OrderId:   brokerOrderID,
		})
	} else {
		orderState, err = b.ordersClient.GetOrderState(ctx, &investapi.GetOrderStateRequest{
			AccountId: brokerID,
			OrderId:   brokerOrderID,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка получения ордера: %w", err)
	}

	order := b.mapOrderState(orderState, accountID)
	return &order, nil
}

// mapOrderState маппинг состояния ордера
func (b *TinkoffBroker) mapOrderState(orderState *investapi.OrderState, accountID string) Order {
	order := Order{
		ID:             "tinkoff_order_" + orderState.OrderId,
		BrokerOrderID:  orderState.OrderId,
		AccountID:      accountID,
		InstrumentID:   "tinkoff_" + orderState.InstrumentUid,
		Direction:      mapOrderDirection(orderState.Direction),
		OrderType:      mapOrderType(orderState.OrderType),
		Status:         mapOrderStatus(orderState.ExecutionReportStatus),
		Quantity:       orderState.LotsRequested,
		QuantityFilled: orderState.LotsExecuted,
		CreatedAt:      orderState.OrderDate.AsTime(),
		UpdatedAt:      time.Now(),
	}

	if orderState.InitialOrderPrice != nil {
		order.Price = moneyValueToFloat(orderState.InitialOrderPrice)
	}

	if orderState.AveragePositionPrice != nil {
		order.AvgFillPrice = moneyValueToFloat(orderState.AveragePositionPrice)
	}

	if orderState.InitialCommission != nil {
		order.Commission = moneyValueToFloat(orderState.InitialCommission)
	}

	return order
}

// GetMarginInfo получает маржинальную информацию
func (b *TinkoffBroker) GetMarginInfo(ctx context.Context, accountID string) (*MarginInfo, error) {
	if !b.connected {
		return nil, ErrNotConnected
	}

	brokerID, err := b.getBrokerAccountID(accountID)
	if err != nil {
		return nil, err
	}

	resp, err := b.usersClient.GetMarginAttributes(ctx, &investapi.GetMarginAttributesRequest{
		AccountId: brokerID,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения маржинальной информации: %w", err)
	}

	marginInfo := &MarginInfo{
		LiquidPortfolio:      moneyValueToFloat(resp.LiquidPortfolio),
		StartingMargin:       moneyValueToFloat(resp.StartingMargin),
		MinimalMargin:        moneyValueToFloat(resp.MinimalMargin),
		AmountOfMissingFunds: moneyValueToFloat(resp.AmountOfMissingFunds),
		Currency:             resp.LiquidPortfolio.Currency,
	}

	if resp.FundsSufficiencyLevel != nil {
		marginInfo.FundsSufficiency = quotationToFloat(resp.FundsSufficiencyLevel)
	}

	return marginInfo, nil
}

// getBrokerAccountID получает ID счета в системе Тинькофф по внутреннему ID
func (b *TinkoffBroker) getBrokerAccountID(internalID string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if brokerID, ok := b.accountMap[internalID]; ok {
		return brokerID, nil
	}

	return "", ErrAccountNotFound
}

// ============================================================================
// Заглушка для MOEX
// ============================================================================

// MOEXBroker заглушка для будущей реализации MOEX
type MOEXBroker struct {
	config    BrokerConfig
	logger    *slog.Logger
	connected bool
}

// NewMOEXBroker создает нового MOEX брокера (заглушка)
func NewMOEXBroker(config BrokerConfig) (*MOEXBroker, error) {
	logger := slog.Default().With(
		"broker", "moex",
		"sandbox", config.Sandbox,
	)

	logger.Warn("MOEX брокер находится в разработке, используется заглушка")

	return &MOEXBroker{
		config: config,
		logger: logger,
	}, nil
}

// Name возвращает название брокера
func (b *MOEXBroker) Name() string {
	return "MOEX (заглушка)"
}

// IsAvailable проверяет доступность
func (b *MOEXBroker) IsAvailable() bool {
	return b.connected
}

// Connect устанавливает соединение
func (b *MOEXBroker) Connect(ctx context.Context) error {
	b.logger.Info("MOEX брокер: Connect (заглушка)")
	b.connected = true
	return nil
}

// Disconnect закрывает соединение
func (b *MOEXBroker) Disconnect() error {
	b.logger.Info("MOEX брокер: Disconnect (заглушка)")
	b.connected = false
	return nil
}

// Close закрывает брокера
func (b *MOEXBroker) Close() error {
	return b.Disconnect()
}

// GetInstruments получает инструменты (заглушка)
func (b *MOEXBroker) GetInstruments(ctx context.Context) ([]Instrument, error) {
	return []Instrument{}, nil
}

// GetInstrument получает инструмент по ID (заглушка)
func (b *MOEXBroker) GetInstrument(ctx context.Context, id string) (*Instrument, error) {
	return nil, ErrNotImplemented
}

// SearchInstruments ищет инструменты (заглушка)
func (b *MOEXBroker) SearchInstruments(ctx context.Context, query string) ([]Instrument, error) {
	return nil, ErrNotImplemented
}

// GetAccounts получает счета (заглушка)
func (b *MOEXBroker) GetAccounts(ctx context.Context) ([]Account, error) {
	return []Account{
		{
			ID:          "moex_demo_1",
			BrokerID:    "demo_1",
			Broker:      "moex",
			Name:        "Демо-счет MOEX",
			Type:        AccountTypeBroker,
			Status:      AccountStatusOpen,
			OpenedAt:    time.Now(),
			AccessLevel: AccessLevelFull,
		},
	}, nil
}

// GetPortfolio получает портфель (заглушка)
func (b *MOEXBroker) GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error) {
	return &Portfolio{
		AccountID:  accountID,
		TotalValue: NewMoneyAmountFromFloat("RUB", 1000000),
		Positions:  []Position{},
		UpdatedAt:  time.Now(),
	}, nil
}

// GetPositions получает позиции (заглушка)
func (b *MOEXBroker) GetPositions(ctx context.Context, accountID string) ([]Position, error) {
	return []Position{}, nil
}

// GetBalance получает баланс (заглушка)
func (b *MOEXBroker) GetBalance(ctx context.Context, accountID string) (map[string]float64, error) {
	return map[string]float64{"RUB": 1000000}, nil
}

// GetCandles получает свечи (заглушка)
func (b *MOEXBroker) GetCandles(ctx context.Context, instrumentID string, from, to time.Time, interval CandleInterval) ([]Candle, error) {
	return []Candle{}, nil
}

// GetLastPrice получает последнюю цену (заглушка)
func (b *MOEXBroker) GetLastPrice(ctx context.Context, instrumentID string) (*LastPrice, error) {
	return &LastPrice{
		InstrumentID: instrumentID,
		Price:        100.0,
		Timestamp:    time.Now(),
	}, nil
}

// GetLastPrices получает последние цены (заглушка)
func (b *MOEXBroker) GetLastPrices(ctx context.Context, instrumentIDs []string) ([]LastPrice, error) {
	var prices []LastPrice
	for _, id := range instrumentIDs {
		prices = append(prices, LastPrice{
			InstrumentID: id,
			Price:        100.0,
			Timestamp:    time.Now(),
		})
	}
	return prices, nil
}

// GetOrderBook получает стакан (заглушка)
func (b *MOEXBroker) GetOrderBook(ctx context.Context, instrumentID string, depth int) (*OrderBook, error) {
	return &OrderBook{
		InstrumentID: instrumentID,
		Depth:        depth,
		Bids:         []OrderBookLevel{},
		Asks:         []OrderBookLevel{},
		Timestamp:    time.Now(),
		Exchange:     "MOEX",
	}, nil
}

// PlaceOrder размещает ордер (заглушка)
func (b *MOEXBroker) PlaceOrder(ctx context.Context, order Order) (*OrderResult, error) {
	return &OrderResult{
		OrderID:        "moex_order_" + time.Now().Format("20060102150405"),
		BrokerOrderID:  "order_123",
		Status:         OrderStatusFilled,
		QuantityFilled: order.Quantity,
	}, nil
}

// CancelOrder отменяет ордер (заглушка)
func (b *MOEXBroker) CancelOrder(ctx context.Context, accountID, orderID string) error {
	return nil
}

// GetOrders получает список ордеров (заглушка)
func (b *MOEXBroker) GetOrders(ctx context.Context, accountID string) ([]Order, error) {
	return []Order{}, nil
}

// GetOrder получает информацию об ордере (заглушка)
func (b *MOEXBroker) GetOrder(ctx context.Context, accountID, orderID string) (*Order, error) {
	return nil, ErrNotImplemented
}

// GetMarginInfo получает маржинальную информацию (заглушка)
func (b *MOEXBroker) GetMarginInfo(ctx context.Context, accountID string) (*MarginInfo, error) {
	return &MarginInfo{
		LiquidPortfolio:      1000000,
		StartingMargin:       500000,
		MinimalMargin:        250000,
		FundsSufficiency:     2.0,
		AmountOfMissingFunds: 0,
		Currency:             "RUB",
	}, nil
}

// ============================================================================
// BrokerFactory - фабрика для создания брокеров
// ============================================================================

// BrokerBuilder интерфейс для строителей брокеров
type BrokerBuilder interface {
	Build(config BrokerConfig) (Broker, error)
}

// TinkoffBrokerBuilder строитель для Тинькофф брокера
type TinkoffBrokerBuilder struct{}

// Build создает экземпляр Тинькофф брокера
func (b *TinkoffBrokerBuilder) Build(config BrokerConfig) (Broker, error) {
	return NewTinkoffBroker(config)
}

// MOEXBrokerBuilder строитель для MOEX брокера
type MOEXBrokerBuilder struct{}

// Build создает экземпляр MOEX брокера
func (b *MOEXBrokerBuilder) Build(config BrokerConfig) (Broker, error) {
	return NewMOEXBroker(config)
}

// BrokerFactory фабрика для создания брокеров
type BrokerFactory struct {
	builders map[string]BrokerBuilder
	logger   *slog.Logger
}

// NewBrokerFactory создает новую фабрику брокеров
func NewBrokerFactory() *BrokerFactory {
	return &BrokerFactory{
		builders: make(map[string]BrokerBuilder),
		logger:   slog.Default().With("component", "broker_factory"),
	}
}

// Register регистрирует строителя для типа брокера
func (f *BrokerFactory) Register(brokerType string, builder BrokerBuilder) {
	f.builders[brokerType] = builder
	f.logger.Info("зарегистрирован брокер", "type", brokerType)
}

// Create создает брокера указанного типа
func (f *BrokerFactory) Create(brokerType string, config BrokerConfig) (Broker, error) {
	builder, ok := f.builders[brokerType]
	if !ok {
		return nil, fmt.Errorf("неизвестный тип брокера: %s", brokerType)
	}

	f.logger.Info("создание брокера", "type", brokerType, "sandbox", config.Sandbox)
	return builder.Build(config)
}

// ============================================================================
// Примеры использования
// ============================================================================

// runExamplePortfolioMonitor пример мониторинга портфеля
func runExamplePortfolioMonitor(ctx context.Context, broker Broker, accountID string) {
	logger := slog.Default().With("example", "portfolio_monitor")

	// Получаем портфель
	portfolio, err := broker.GetPortfolio(ctx, accountID)
	if err != nil {
		logger.Error("ошибка получения портфеля", "error", err)
		return
	}

	logger.Info("портфель",
		"total_value", portfolio.TotalValue.String(),
		"positions_count", len(portfolio.Positions),
	)

	// Выводим позиции
	for i, pos := range portfolio.Positions {
		logger.Info(fmt.Sprintf("позиция %d", i+1),
			"instrument", pos.InstrumentID,
			"quantity", pos.Quantity,
			"current_price", pos.CurrentPrice.String(),
			"profit_loss", fmt.Sprintf("%.2f", pos.ProfitLoss),
			"profit_loss_percent", fmt.Sprintf("%.2f%%", pos.ProfitLossPercent),
		)
	}
}

// runExampleMarketData пример получения рыночных данных
func runExampleMarketData(ctx context.Context, broker Broker, instrumentID string) {
	logger := slog.Default().With("example", "market_data")

	// Получаем последнюю цену
	lastPrice, err := broker.GetLastPrice(ctx, instrumentID)
	if err != nil {
		logger.Error("ошибка получения последней цены", "error", err)
	} else {
		logger.Info("последняя цена", "price", lastPrice.Price)
	}

	// Получаем стакан
	orderBook, err := broker.GetOrderBook(ctx, instrumentID, 10)
	if err != nil {
		logger.Error("ошибка получения стакана", "error", err)
	} else {
		logger.Info("стакан",
			"bids_count", len(orderBook.Bids),
			"asks_count", len(orderBook.Asks),
		)
	}

	// Получаем свечи
	to := time.Now()
	from := to.AddDate(0, 0, -7) // 7 дней назад

	candles, err := broker.GetCandles(ctx, instrumentID, from, to, CandleInterval1Day)
	if err != nil {
		logger.Error("ошибка получения свечей", "error", err)
	} else {
		logger.Info("свечи", "count", len(candles))
		if len(candles) > 0 {
			last := candles[len(candles)-1]
			logger.Info("последняя свеча",
				"open", last.Open,
				"high", last.High,
				"low", last.Low,
				"close", last.Close,
				"volume", last.Volume,
			)
		}
	}
}

// runExampleTrading пример торговых операций
func runExampleTrading(ctx context.Context, broker Broker, accountID, instrumentID string) {
	logger := slog.Default().With("example", "trading")

	// Получаем текущие ордера
	orders, err := broker.GetOrders(ctx, accountID)
	if err != nil {
		logger.Error("ошибка получения ордеров", "error", err)
	} else {
		logger.Info("активные ордера", "count", len(orders))
	}

	// Получаем последнюю цену для примера
	lastPrice, err := broker.GetLastPrice(ctx, instrumentID)
	if err != nil {
		logger.Error("ошибка получения цены", "error", err)
		return
	}

	// Создаем лимитный ордер на покупку (по цене на 1% ниже рынка)
	order := Order{
		AccountID:    accountID,
		InstrumentID: instrumentID,
		Direction:    OrderDirectionBuy,
		OrderType:    OrderTypeLimit,
		Quantity:     1,
		Price:        lastPrice.Price * 0.99,
	}

	logger.Info("размещение ордера",
		"instrument", instrumentID,
		"price", order.Price,
		"quantity", order.Quantity,
	)

	result, err := broker.PlaceOrder(ctx, order)
	if err != nil {
		logger.Error("ошибка размещения ордера", "error", err)
		return
	}

	logger.Info("ордер размещен",
		"order_id", result.OrderID,
		"status", result.Status,
	)

	// Ждем немного и отменяем ордер
	time.Sleep(2 * time.Second)

	err = broker.CancelOrder(ctx, accountID, result.OrderID)
	if err != nil {
		logger.Error("ошибка отмены ордера", "error", err)
	} else {
		logger.Info("ордер отменен", "order_id", result.OrderID)
	}
}

// runExampleMultiBroker пример работы с несколькими брокерами
func runExampleMultiBroker(ctx context.Context, brokers []Broker) {
	logger := slog.Default().With("example", "multi_broker")

	for _, b := range brokers {
		logger.Info("работа с брокером", "name", b.Name())

		// Получаем счета
		accounts, err := b.GetAccounts(ctx)
		if err != nil {
			logger.Error("ошибка получения счетов", "broker", b.Name(), "error", err)
			continue
		}

		logger.Info("счета", "broker", b.Name(), "count", len(accounts))

		// Получаем баланс по каждому счету
		for _, acc := range accounts {
			balance, err := b.GetBalance(ctx, acc.ID)
			if err != nil {
				logger.Error("ошибка получения баланса", "broker", b.Name(), "account", acc.ID, "error", err)
				continue
			}

			logger.Info("баланс",
				"broker", b.Name(),
				"account", acc.Name,
				"balance", balance,
			)
		}
	}
}

// ============================================================================
// Основная функция
// ============================================================================

func main() {
	// Настройка логирования
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("запуск Unified Broker API")

	// Создаем контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("получен сигнал завершения")
		cancel()
	}()

	// Создаем фабрику брокеров
	factory := NewBrokerFactory()

	// Регистрируем доступных брокеров
	factory.Register("tinkoff", &TinkoffBrokerBuilder{})
	factory.Register("moex", &MOEXBrokerBuilder{})

	// Конфигурация для Тинькофф (режим песочницы для тестирования)
	tinkoffConfig := DefaultConfig()
	tinkoffConfig.APIKey = os.Getenv("TINKOFF_TOKEN") // Берем токен из переменной окружения
	tinkoffConfig.Sandbox = true
	tinkoffConfig.RateLimit = 10
	tinkoffConfig.Timeout = 30 * time.Second

	// Если токен не задан, используем демо-режим
	if tinkoffConfig.APIKey == "" {
		logger.Warn("TINKOFF_TOKEN не задан, используется демо-режим с заглушкой")
		tinkoffConfig.APIKey = "demo_token"
	}

	// Создаем Тинькофф брокера
	tinkoffBroker, err := factory.Create("tinkoff", tinkoffConfig)
	if err != nil {
		logger.Error("ошибка создания Тинькофф брокера", "error", err)
		return
	}

	// Конфигурация для MOEX (заглушка)
	moexConfig := DefaultConfig()
	moexConfig.Sandbox = true

	moexBroker, err := factory.Create("moex", moexConfig)
	if err != nil {
		logger.Error("ошибка создания MOEX брокера", "error", err)
		return
	}

	// Подключаемся к брокерам
	if err := tinkoffBroker.Connect(ctx); err != nil {
		logger.Error("ошибка подключения к Тинькофф", "error", err)
		return
	}
	defer tinkoffBroker.Close()

	if err := moexBroker.Connect(ctx); err != nil {
		logger.Error("ошибка подключения к MOEX", "error", err)
		return
	}
	defer moexBroker.Close()

	// Получаем счета Тинькофф
	accounts, err := tinkoffBroker.GetAccounts(ctx)
	if err != nil {
		logger.Error("ошибка получения счетов Тинькофф", "error", err)
		return
	}

	if len(accounts) == 0 {
		logger.Error("нет доступных счетов")
		return
	}

	accountID := accounts[0].ID
	logger.Info("используется счет", "id", accountID, "name", accounts[0].Name)

	// Получаем инструменты
	instruments, err := tinkoffBroker.GetInstruments(ctx)
	if err != nil {
		logger.Error("ошибка получения инструментов", "error", err)
	} else {
		logger.Info("получены инструменты", "count", len(instruments))

		// Ищем SBER для примеров
		var sberInstrument *Instrument
		for _, instr := range instruments {
			if instr.Ticker == "SBER" && instr.Type == InstrumentTypeStock {
				sberInstrument = &instr
				break
			}
		}

		if sberInstrument != nil {
			logger.Info("найден инструмент SBER", "id", sberInstrument.ID)

			// Запускаем примеры
			logger.Info("=== Пример мониторинга портфеля ===")
			runExamplePortfolioMonitor(ctx, tinkoffBroker, accountID)

			logger.Info("=== Пример получения рыночных данных ===")
			runExampleMarketData(ctx, tinkoffBroker, sberInstrument.ID)

			logger.Info("=== Пример торговых операций ===")
			runExampleTrading(ctx, tinkoffBroker, accountID, sberInstrument.ID)
		}
	}

	// Пример работы с несколькими брокерами
	logger.Info("=== Пример работы с несколькими брокерами ===")
	runExampleMultiBroker(ctx, []Broker{tinkoffBroker, moexBroker})

	logger.Info("программа завершена")
}
