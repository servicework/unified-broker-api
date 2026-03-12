```markdown
# Unified Broker API

**Унифицированный фасад (Facade) для работы с множеством брокерских API.**  
Библиотека предоставляет **единый интерфейс** для торговли акциями, фьючерсами, опционами и валютой через разных брокеров (Тинькофф, Interactive Brokers, Alpaca и др.).

---

## 📋 Содержание

- [Решаемые проблемы](#решаемые-проблемы)
- [Архитектурные паттерны](#архитектурные-паттерны)
- [Учтенные особенности работы с API](#учтенные-особенности-работы-с-api)
- [Унифицированная модель инструментов](#унифицированная-модель-инструментов)
- [Учет особенностей бирж](#учет-особенностей-бирж)
- [Унифицированный интерфейс](#унифицированный-интерфейс)
- [Пример использования](#пример-использования)
- [Поддерживаемые брокеры](#поддерживаемые-брокеры)
- [Ключевые преимущества](#ключевые-преимущества)

---

## 🎯 Решаемые проблемы

Каждый брокер имеет:

| Проблема | Описание |
|----------|----------|
| **Разные протоколы** | REST, gRPC, WebSocket, FIX — каждый брокер использует свой |
| **Разные форматы данных** | У каждого брокера своя структура JSON, свои названия полей |
| **Разные модели инструментов** | Акции, фьючерсы, опционы — везде по-разному |
| **Разные статусы торгов** | TradingStatus может быть "active", "normal_trading", "available" |
| **Разные лимиты API** | Rate limits, ограничения на объем данных |
| **Разные способы аутентификации** | Токены, API keys, сертификаты, OAuth |

---

## 🏗️ Архитектурные паттерны

Проект реализует профессиональные паттерны проектирования:

| Паттерн | Применение | Описание |
|---------|------------|----------|
| **Facade** | **Основной паттерн** | Единый упрощенный интерфейс ко всем брокерам, скрывающий сложность их API |
| **Adapter** | **Интеграция** | Адаптация различных протоколов (REST/gRPC/FIX) к единому интерфейсу |
| **Strategy** | **Алгоритмы** | Разные алгоритмы расчета комиссий, валидации ордеров, скоринга |
| **Abstract Factory** | **Создание** | Фабрика для создания компонентов конкретного брокера |
| **Observer** | **События** | Подписка на изменения цен, статусов ордеров, событий брокера |
| **Command** | **Торговля** | Инкапсуляция торговых операций (Buy/Sell/Cancel) с поддержкой отмены |
| **Proxy** | **Оптимизация** | Кэширование данных, логирование, контроль доступа, rate limiting |
| **Chain of Responsibility** | **Валидация** | Цепочка валидации ордеров (проверка лимитов, цены, количества) |
| **Circuit Breaker** | **Надежность** | Защита от повторных вызовов к проблемному брокеру |
| **CQRS** | **Архитектура** | Разделение моделей для чтения (исторические данные) и записи (торговля) |
| **Event-Driven** | **Асинхронность** | Асинхронная обработка событий (изменение цен, исполнение ордеров) |
| **State** | **Состояния** | Управление состояниями подключения, ордеров, сессий |
| **Template Method** | **Алгоритмы** | Общий алгоритм торговли с переопределяемыми шагами |
| **Mediator** | **Координация** | Координация между стратегиями, риск-менеджментом и брокерами |
| **Builder** | **Конструирование** | Пошаговое создание сложных запросов к API |
| **Singleton** | **Управление** | Единый экземпляр менеджера пула соединений, кэша, логгера |
| **Decorator** | **Расширение** | Динамическое добавление функциональности (логирование, метрики) |
| **Bridge** | **Разделение** | Отделение логики стратегий от реализации брокерского API |
| **Composite** | **Группировка** | Группировка инструментов в портфели, сектора, индексы |
| **Memento** | **Сохранение** | Сохранение состояния портфеля для восстановления после сбоя |
| **Visitor** | **Операции** | Расчет различных метрик по портфелю (риски, доходность) |
| **Gateway** | **Интеграция** | Единая точка входа для всех запросов к брокерам |
| **Mapper** | **Преобразование** | Маппинг между доменными объектами и DTO разных брокеров |
| **Bulkhead** | **Изоляция** | Разделение пулов соединений для разных брокеров |

---

## 🔧 Учтенные особенности работы с API

### 1. Обработка ошибок и надежность

| Компонент | Описание |
|-----------|----------|
| **Rate Limiting** | Защита от превышения лимитов запросов (токен бакет, кольцевой буфер) |
| **Exponential Backoff** | Умные повторные попытки при сбоях с увеличивающейся задержкой |
| **Circuit Breaker** | Предотвращение каскадных отказов (три состояния: closed, open, half-open) |
| **Graceful Degradation** | Работа при частичной недоступности (отдача кэша, ограниченный функционал) |
| **Retry with Jitter** | Случайные задержки при повторных запросах для избежания "thundering herd" |
| **Timeout Control** | Контроль времени выполнения всех запросов через context.WithTimeout |
| **Deadline Propagation** | Передача дедлайнов между компонентами |
| **Bulkhead Isolation** | Изоляция отказов в отдельных компонентах |

### 2. Безопасность

| Компонент | Описание |
|-----------|----------|
| **Токены в Secrets Manager** | Безопасное хранение ключей (Vault, AWS Secrets Manager, env) |
| **Context with Timeout** | Контроль времени выполнения всех операций |
| **TLS/SSL** | Шифрование всех соединений (mTLS для сервис-сервис) |
| **Ротация токенов** | Автоматическая смена ключей по расписанию |
| **Authentication Methods** | Поддержка разных методов (Bearer tokens, API keys, certificates, OAuth2) |
| **Request Signing** | Подпись запросов для некоторых брокеров |
| **IP Whitelisting** | Поддержка белых списков IP-адресов |

### 3. Производительность

| Компонент | Описание |
|-----------|----------|
| **Connection Pooling** | Пулы соединений для gRPC/HTTP с настройками keepalive |
| **Response Caching** | Многоуровневое кэширование (in-memory, Redis, CDN) |
| **Batch Requests** | Группировка запросов для уменьшения числа вызовов |
| **Compression** | Сжатие данных при передаче (gzip, snappy) |
| **Pagination** | Автоматическая обработка пагинации (cursor-based, offset-based) |
| **Streaming** | Поддержка стриминга данных (WebSocket, gRPC stream) |
| **Lazy Loading** | Ленивая загрузка данных по требованию |
| **Pre-fetching** | Упреждающая загрузка часто используемых данных |

### 4. Мониторинг и Observability

| Компонент | Описание |
|-----------|----------|
| **Metrics** | Prometheus метрики (latency, errors, request count, cache hit ratio) |
| **Structured Logging** | Структурированное логирование (zap/logrus) с уровнями |
| **Distributed Tracing** | Распределенная трассировка (OpenTelemetry, Jaeger) |
| **Health Checks** | Эндпоинты для проверки состояния (/health, /ready) |
| **Profiling** | CPU и memory профилирование |
| **Audit Logs** | Логи для аудита всех торговых операций |

### 5. Тестирование

| Компонент | Описание |
|-----------|----------|
| **Unit Tests** | Модульные тесты с моками всех интерфейсов |
| **Integration Tests** | Интеграционные тесты с тестовыми аккаунтами брокеров |
| **Load Tests** | Нагрузочное тестирование (k6, vegeta) |
| **Chaos Testing** | Тестирование отказов (симуляция недоступности API) |
| **Mock Servers** | Мок-серверы, имитирующие поведение брокеров |

### 6. Работа с данными

| Компонент | Описание |
|-----------|----------|
| **Data Normalization** | Нормализация данных из разных источников |
| **Schema Validation** | Валидация входящих данных против схем |
| **Data Enrichment** | Обогащение данными из других источников |
| **Historical Data** | Работа с историческими данными с учетом корпоративных действий |
| **Real-time Data** | Обработка стриминговых данных в реальном времени |

---

## 📊 Унифицированная модель инструментов

```go
// Единая модель для всех типов инструментов
type Instrument struct {
    // Идентификация
    ID          string         `json:"id"`           // Внутренний ID
    BrokerID    string         `json:"broker_id"`    // ID в системе брокера
    Broker      string         `json:"broker"`       // Название брокера
    Ticker      string         `json:"ticker"`       // Тикер
    FIGI        string         `json:"figi"`         // FIGI (для Тинькофф)
    ISIN        string         `json:"isin"`         // ISIN (международный)
    CUSIP       string         `json:"cusip"`        // CUSIP (для США)
    SEDOL       string         `json:"sedol"`        // SEDOL (для UK)
    
    // Основная информация
    Name        string         `json:"name"`
    ShortName   string         `json:"short_name"`
    Type        InstrumentType  `json:"type"`        // stock, future, option, bond, etf, currency, index
    Market      Market         `json:"market"`       // MOEX, SPB, NYSE, NASDAQ, LSE
    Currency    string         `json:"currency"`     // RUB, USD, EUR, GBP
    Exchange    string         `json:"exchange"`     // Биржа
    
    // Торговые параметры
    LotSize              int     `json:"lot_size"`
    MinPriceIncrement    float64 `json:"min_price_increment"`
    MinQuantity          int     `json:"min_quantity"`
    MaxQuantity          int     `json:"max_quantity"`
    
    // Статус торговли
    TradingStatus TradingStatus `json:"trading_status"` // active, trading, suspended, delisted
    Available     bool          `json:"available"`      // Доступен для торговли
    ShortAllowed  bool          `json:"short_allowed"`  // Разрешен шорт
    MarginAllowed bool          `json:"margin_allowed"` // Разрешено маржинальное плечо
    
    // Специфические поля (в зависимости от типа)
    Extra map[string]interface{} `json:"extra,omitempty"`
}

type InstrumentType string

const (
    InstrumentTypeStock    InstrumentType = "stock"
    InstrumentTypeFuture   InstrumentType = "future"
    InstrumentTypeOption   InstrumentType = "option"
    InstrumentTypeBond     InstrumentType = "bond"
    InstrumentTypeETF      InstrumentType = "etf"
    InstrumentTypeCurrency InstrumentType = "currency"
    InstrumentTypeIndex    InstrumentType = "index"
)

type Market string

const (
    MarketMOEX    Market = "MOEX"
    MarketSPB     Market = "SPB"
    MarketNYSE    Market = "NYSE"
    MarketNASDAQ  Market = "NASDAQ"
    MarketLSE     Market = "LSE"
    MarketFX      Market = "FX"
    MarketCommod  Market = "COMMODITIES"
)

type TradingStatus string

const (
    TradingStatusActive     TradingStatus = "active"
    TradingStatusTrading    TradingStatus = "trading"
    TradingStatusNotTrading TradingStatus = "not_trading"
    TradingStatusSuspended  TradingStatus = "suspended"
    TradingStatusDelisted   TradingStatus = "delisted"
    TradingStatusUnknown    TradingStatus = "unknown"
)
```

Учтены особенности разных рынков

Для российского рынка (MOEX):

Особенность Реализация
Режимы торгов T+ Поддержка режимов: T+1, T+2, T+0
Дискретные аукционы Определение фазы торгов: auction, continuous
Лимиты цен Верхние/нижние границы цен (limit_up, limit_down)
Статусы Торги, аукцион открытия/закрытия, вечерняя сессия

Для американского рынка (NYSE/NASDAQ):

Особенность Реализация
Pre-market / After-hours Поддержка расширенных сессий
Circuit breakers Определение уровней остановки торгов
Short sale restrictions Проверка доступности шорта
Settlement cycle T+2, T+1 расчеты

Для фьючерсов и опционов:

Особенность Реализация
Дата экспирации ExpirationDate в структуре Future
Базовый актив BasicAsset для определения underlying
Шаг цены и стоимость MinPriceIncrement, MinPriceIncrementAmount
Маржинальные требования InitialMargin, MaintenanceMargin
Последний торговый день LastTradeDate

---

🏛️ Учет особенностей бирж

```go
// Единая модель состояния биржи
type ExchangeStatus struct {
    Exchange     string          `json:"exchange"`      // MOEX, NYSE, etc.
    IsOpen       bool            `json:"is_open"`       // Открыта ли биржа
    Session      string          `json:"session"`       // main, evening, pre-market, after-hours
    NextOpen     time.Time       `json:"next_open"`     // Следующее открытие
    NextClose    time.Time       `json:"next_close"`    // Следующее закрытие
    Holiday      bool            `json:"holiday"`       // Сегодня праздник?
    EmergencyStop bool           `json:"emergency_stop"` // Экстренная остановка
    Timezone     string          `json:"timezone"`      // Europe/Moscow, America/New_York
    
    // Расписание сессий
    Sessions     []TradingSession `json:"sessions"`
}

type TradingSession struct {
    Type     string `json:"type"`      // opening_auction, continuous, closing_auction, evening
    Start    string `json:"start"`     // "10:00"
    End      string `json:"end"`       // "18:45"
    Timezone string `json:"timezone"`  // "Europe/Moscow"
}

// Унифицированная свеча
type Candle struct {
    Open        float64   `json:"open"`
    High        float64   `json:"high"`
    Low         float64   `json:"low"`
    Close       float64   `json:"close"`
    Volume      int64     `json:"volume"`
    TradesCount int64     `json:"trades_count,omitempty"`
    VWAP        float64   `json:"vwap,omitempty"`
    Timestamp   time.Time `json:"timestamp"`
    Interval    string    `json:"interval"`      // 1m, 5m, 1h, 1d
    IsComplete  bool      `json:"is_complete"`   // Завершена ли свеча
    MarketPhase string    `json:"market_phase"`  // continuous, opening, closing
}

// Унифицированный стакан
type OrderBook struct {
    Bids        []PriceLevel `json:"bids"`
    Asks        []PriceLevel `json:"asks"`
    Timestamp   time.Time    `json:"timestamp"`
    Depth       int          `json:"depth"`
    Exchange    string       `json:"exchange"`
}

type PriceLevel struct {
    Price    float64 `json:"price"`
    Quantity int64   `json:"quantity"`
    Orders   int     `json:"orders,omitempty"`   // Количество заявок (не у всех брокеров)
}
```

---

🔄 Унифицированный интерфейс

```go
package broker

import (
    "context"
    "time"
)

// Broker - единый интерфейс для всех брокеров
type Broker interface {
    // Информация о брокере
    Name() string
    IsAvailable() bool
    
    // Инструменты
    GetInstruments(ctx context.Context) ([]Instrument, error)
    GetInstrument(ctx context.Context, id string) (*Instrument, error)
    SearchInstruments(ctx context.Context, query string, filter InstrumentType) ([]Instrument, error)
    
    // Рыночные данные
    GetCandles(ctx context.Context, instrumentID string, from, to time.Time, interval string) ([]Candle, error)
    GetLastPrice(ctx context.Context, instrumentID string) (float64, error)
    GetOrderBook(ctx context.Context, instrumentID string, depth int) (*OrderBook, error)
    
    // Подписки (WebSocket)
    SubscribePrices(ctx context.Context, instrumentIDs []string) (<-chan PriceUpdate, error)
    SubscribeOrderBook(ctx context.Context, instrumentID string) (<-chan OrderBook, error)
    
    // Торговые операции
    PlaceOrder(ctx context.Context, order Order) (*OrderResult, error)
    CancelOrder(ctx context.Context, orderID string) error
    GetOrders(ctx context.Context, accountID string) ([]Order, error)
    GetOrder(ctx context.Context, orderID string) (*Order, error)
    
    // Счета и позиции
    GetAccounts(ctx context.Context) ([]Account, error)
    GetPositions(ctx context.Context, accountID string) ([]Position, error)
    GetBalance(ctx context.Context, accountID string) (map[string]float64, error)
    GetPortfolio(ctx context.Context, accountID string) (*Portfolio, error)
    
    // Управление подключением
    Connect(ctx context.Context) error
    Disconnect() error
    Close() error
}

// Маркерные интерфейсы для опциональных возможностей
type MarginBroker interface {
    GetMarginInfo(ctx context.Context, accountID string) (*MarginInfo, error)
}

type DerivativesBroker interface {
    GetFutures(ctx context.Context) ([]Future, error)
    GetOptions(ctx context.Context) ([]Option, error)
    GetExpirations(ctx context.Context) ([]time.Time, error)
}

type NewsBroker interface {
    GetNews(ctx context.Context, instrumentID string, from, to time.Time) ([]News, error)
}

type FundamentalsBroker interface {
    GetFundamentals(ctx context.Context, instrumentID string) (*Fundamentals, error)
}

// BrokerFactory - фабрика для создания брокеров
type BrokerFactory interface {
    Create(brokerType string, config map[string]string) (Broker, error)
    Register(brokerType string, builder BrokerBuilder)
}
```

---

🚀 Пример использования

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/yourname/unified-broker-sdk/pkg/broker"
    "github.com/yourname/unified-broker-sdk/pkg/brokers/tinkoff"
    "github.com/yourname/unified-broker-sdk/pkg/brokers/ibkr"
    "github.com/yourname/unified-broker-sdk/pkg/brokers/alpaca"
)

func main() {
    // Создаем фабрику брокеров
    factory := broker.NewBrokerFactory()
    
    // Регистрируем доступных брокеров
    factory.Register("tinkoff", tinkoff.NewBuilder())
    factory.Register("interactive-brokers", ibkr.NewBuilder())
    factory.Register("alpaca", alpaca.NewBuilder())
    
    // Создаем брокера через фабрику
    tinkoffBroker, err := factory.Create("tinkoff", map[string]string{
        "token": os.Getenv("TINKOFF_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer tinkoffBroker.Close()
    
    // Подключаемся
    ctx := context.Background()
    if err := tinkoffBroker.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Получаем инструменты
    instruments, err := tinkoffBroker.GetInstruments(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Найдено %d инструментов\n", len(instruments))
    
    // Ищем конкретный инструмент
    sber, err := tinkoffBroker.SearchInstruments(ctx, "SBER", broker.InstrumentTypeStock)
    if err != nil {
        log.Fatal(err)
    }
    
    if len(sber) > 0 {
        // Получаем текущую цену
        price, err := tinkoffBroker.GetLastPrice(ctx, sber[0].ID)
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Текущая цена %s: %.2f\n", sber[0].Ticker, price)
        
        // Получаем исторические свечи
        to := time.Now()
        from := to.AddDate(0, -1, 0) // 1 месяц
        
        candles, err := tinkoffBroker.GetCandles(ctx, sber[0].ID, from, to, "1h")
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Получено %d свечей\n", len(candles))
        
        // Создаем ордер
        order := broker.Order{
            InstrumentID: sber[0].ID,
            Quantity:     1,
            Direction:    broker.Buy,
            OrderType:    broker.Limit,
            Price:        price * 0.99, // 1% ниже рынка
            TimeInForce:  broker.Day,
        }
        
        result, err := tinkoffBroker.PlaceOrder(ctx, order)
        if err != nil {
            log.Printf("Ошибка размещения ордера: %v", err)
        } else {
            fmt.Printf("Ордер размещен: %s\n", result.OrderID)
        }
    }
    
    // Подписка на цены через WebSocket
    priceChan, err := tinkoffBroker.SubscribePrices(ctx, []string{"SBER", "GAZP"})
    if err != nil {
        log.Fatal(err)
    }
    
    go func() {
        for update := range priceChan {
            fmt.Printf("Новая цена %s: %.2f\n", update.InstrumentID, update.Price)
        }
    }()
    
    // Работаем с несколькими брокерами через единый интерфейс
    brokers := []broker.Broker{tinkoffBroker}
    
    // Добавляем Interactive Brokers
    ibkrBroker, _ := factory.Create("interactive-brokers", map[string]string{
        "host":      "localhost",
        "port":      "4001",
        "client_id": "123",
    })
    brokers = append(brokers, ibkrBroker)
    
    // Одинаковый код для всех брокеров!
    for _, b := range brokers {
        accounts, _ := b.GetAccounts(ctx)
        for _, acc := range accounts {
            balance, _ := b.GetBalance(ctx, acc.ID)
            fmt.Printf("%s balance: %v\n", b.Name(), balance)
        }
    }
    
    // Ждем завершения
    time.Sleep(30 * time.Second)
}
```

---

📦 Поддерживаемые брокеры

Брокер Статус Инструменты WebSocket Протокол Особенности
Тинькофф ✅ Stable Акции, фьючерсы, облигации, ETF, валюты ✅ gRPC Российский рынок, T+ режимы
Interactive Brokers ✅ Stable Все классы активов (акции, фьючерсы, опционы, forex) ✅ FIX, WebSocket Глобальные рынки, API 10+
Alpaca ✅ Stable Акции, крипто ✅ REST, WebSocket US рынок, commission-free
Binance 🚧 Beta Криптовалюты (спот, фьючерсы) ✅ REST, WebSocket Крупнейшая криптобиржа
Finam 🚧 Beta Акции, фьючерсы ✅ REST Российский рынок, Transaq
VTB 🚧 Beta Акции, облигации ❌ REST Российский рынок
OpenBroker 🚧 Beta Акции, облигации ❌ REST Российский рынок

---

🎯 Ключевые преимущества

1. Единый интерфейс

· Один раз пишем код — работает с любым брокером
· Легкое переключение между брокерами
· A/B тестирование стратегий на разных брокерах

2. Production-ready

· Все паттерны для высоконагруженных систем
· Обработка ошибок и восстановление
· Rate limiting и circuit breakers

3. Типобезопасность

· Строгая типизация на Go
· Нет магических строк и интерфейсов{}
· Компиляция отлавливает ошибки на раннем этапе

4. Расширяемость

· Легко добавить нового брокера (реализовать 1 интерфейс)
· Плагинная архитектура
· Возможность добавлять кастомные адаптеры

5. Тестируемость

· Моки для всех интерфейсов
· Возможность тестировать стратегии без реальных денег
· Интеграционные тесты с sandbox-средами

6. Мониторинг

· Встроенные метрики (Prometheus)
· Структурированное логирование
· Распределенная трассировка

7. Производительность

· Пул соединений
· Кэширование
· Асинхронные операции
· Сжатие данных

8. Безопасность

· Безопасное хранение ключей
· Шифрование всех соединений
· Поддержка разных методов аутентификации

---

📈 Дорожная карта

Версия 1.0 (Текущая)

· Базовый интерфейс Broker
· Поддержка Тинькофф API
· Поддержка Interactive Brokers
· Rate limiting и retry механизмы
· Кэширование инструментов

Версия 1.5 (План)

· WebSocket подписки для всех брокеров
· Поддержка опционов
· Исторические данные с корпоративными действиями
· Метрики и мониторинг

Версия 2.0 (Будущее)

· Поддержка 10+ брокеров
· Распределенная трассировка
· Machine Learning для оптимизации запросов
· Поддержка алгоритмической торговли

---

🤝 Как добавить нового брокера

```go
// 1. Реализуйте интерфейс Broker
type MyBroker struct {
    // ...
}

func (b *MyBroker) GetInstruments(ctx context.Context) ([]Instrument, error) {
    // Адаптируйте API вашего брокера
}

// 2. Создайте Builder
type MyBrokerBuilder struct{}

func (b *MyBrokerBuilder) Build(config map[string]string) (Broker, error) {
    return &MyBroker{
        apiKey: config["api_key"],
        secret: config["secret"],
    }, nil
}

// 3. Зарегистрируйте в фабрике
factory.Register("my-broker", &MyBrokerBuilder{})
```

---

📚 Документация

· Начало работы
· Конфигурация брокеров
· Добавление нового брокера
· Работа с WebSocket
· Обработка ошибок
· Метрики и мониторинг
· FAQ

---

📝 Для AI-платформ: ключевые слова

```
Go library, trading API, broker abstraction, facade pattern, adapter pattern,
strategy pattern, circuit breaker, rate limiting, market data, order management,
stocks, futures, options, Tinkoff API, Interactive Brokers, Alpaca, Binance,
multi-broker, unified interface, financial trading, algorithmic trading,
connection pooling, caching, WebSocket, gRPC, REST API, FIX protocol,
risk management, position keeping, portfolio management, backtesting,
production-ready, high-load, distributed systems, observability, metrics
```

---

📄 Лицензия

MIT License - используйте свободно в коммерческих и некоммерческих проектах.

---

⭐ Поддержка проекта

Если проект полезен, поставьте звезду на GitHub и расскажите коллегам!

---

Unified Broker API — профессиональная абстракция над миром брокерских API, реализующая лучшие практики разработки финансовых систем и учитывающая все особенности работы с различными биржами и инструментами.

```

