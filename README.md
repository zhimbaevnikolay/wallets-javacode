# Wallets API Service

## Описание
Тестовое задание от Жимбаева Николая

API для системы кошельков с возможностью обновления баланса и получения текущего состояния баланса.

## Стек
- go
- docker
- docker-compose
- Redis
- PostgreSQL

## Установка и запуск
1. Клонировать репозиторий

`git clone https://github.com/zhimbaevnikolay/wallets-javacode.git`

2. Создать и заполнить config.env

3. Запустить с помощью docker-compose

`docker-compose up --build`

## Использование API

### Создание кошелька
**POST**

`/api/v1/wallet/create`

**Тело запроса(опционально)**

```JSON
{
    "balance": 5000
}
```


**Ответ**
```JSON
{
	"status": "OK",
	"id": "c3f7ab2e-3e0b-4cd0-8f10-f4e751a989a5"
}
```

### Запрос баланса
**GET**

`/api/v1/wallets/{wallet_uuid}`

**Ответ**

```JSON
{
	"status": "OK",
	"balance": 5000
}
```
### Обновление баланса
**POST**

`/api/v1/wallet`

**Тело запроса**

```JSON
{
	"wallet_id": "c3f7ab2e-3e0b-4cd0-8f10-f4e751a989a5",
	"operation_type": "DEPOSIT",
	"amount": 150000
}
```

**Ответ**
```JSON
{
	"ID": "f4eba8a0-ba9a-4f0a-99b8-753bf7908220",
	"WalletID": "c3f7ab2e-3e0b-4cd0-8f10-f4e751a989a5",
	"OperationType": "DEPOSIT",
	"Amount": 150000,
	"Created_at": "2025-03-29T12:22:51.922031Z"
}
```

**Операции**

- `DEPOSIT` - пополнение баланса
- `WITHDRAW` - списание с баланса


