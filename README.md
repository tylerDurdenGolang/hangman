# Лабораторная работа: Разработка распределённых приложений

## Описание проекта

Данный проект представляет собой лабораторную работу по разработке распределённых приложений. Основная цель — создание серверной части для сетевой игры "Виселица" с использованием протокола JSON для обмена данными между сервером и клиентом. Проект реализован на языке Go для сервера и C# для клиентской части.

---

## Цели лабораторной работы

1. Изучить основные принципы разработки клиент-серверных приложений.
2. Реализовать взаимодействие между клиентом и сервером через TCP.
3. Освоить работу с JSON для передачи данных.
4. Организовать корректную обработку ошибок и управление состояниями.

---

## Стек технологий

- **Сервер:**
  - Язык: Go
  - Протокол: TCP
  - Формат сообщений: JSON
  - Архитектура: Чистая архитектура с разделением на слои (контроллеры, репозитории, сервисы)

- **Клиент:**
  - Язык: C#
  - Сериализация/Десериализация JSON: `System.Text.Json`
  - Взаимодействие: TCP

---

## Функциональность

### Сервер

- **Регистрация игрока:** Клиент отправляет имя пользователя, сервер сохраняет его.
- **Создание комнаты:** Игрок может создать комнату, задав её идентификатор и пароль.
- **Присоединение к комнате:** Игрок может присоединиться к существующей комнате.
- **Запуск игры:** Владелец комнаты может запустить игру.
- **Угадывание букв:** Игроки отправляют буквы для угадывания слова.
- **Завершение игры:** Сервер сообщает о выигрыше или проигрыше.

### Клиент

- Отправляет имя игрока на сервер.
- Создаёт комнату с заданными параметрами.
- Отправляет команды в формате JSON (например, `GUESS_LETTER`, `GET_GAME_STATE`).
- Обрабатывает ответы сервера (успех, ошибка, завершение игры).

---

## Структура проекта

### Серверная часть (Go)

├── internal
│   ├── domain          # Основные доменные сущности
│   ├── service         # Логика работы (контроллеры, сервисы)
│   ├── repository      # Хранилище данных (репозитории)
│   └── tcp             # Логика взаимодействия через TCP
└── main.go             # Точка входа