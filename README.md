## Настройка проекта

1. Скопируйте файл `.env.example` из папки `examples` в корень проекта с именем `config.env`:
   ```bash
   cp examples/.env.example config.env

2. Укажите значения для переменных окружения в config.env.

3. Запустите проект: docker-compose --env-file config.env up --build
