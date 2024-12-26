## Настройка проекта

1. Скопируйте файл `config.example.env` в `config.env`:
   ```bash
   cp config.example.env config.env

2. Укажите значения для переменных окружения в config.env.

3. Запустите проект: docker-compose --env-file config.env up --build
