import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import SchedulePage from "./screens/SchedulePage.vue";

export const schedulePageRoute = app.get("/", () => (
  <html lang="ru">
    <head>
      <meta charset="utf-8" />
      <meta
        name="viewport"
        content="width=device-width, initial-scale=1, viewport-fit=cover"
      />
      <title>Расписание — мастерская «Ясна»</title>
      <meta
        name="description"
        content="Кабинет мастера детской столярной мастерской «Ясна»: расписание, журнал, абонементы и клиенты."
      />
      <meta name="theme-color" content="#F5EFE4" />
      <meta name="robots" content="noindex, nofollow" />
      <Styles />
    </head>
    <body class="bg-paper">
      <SchedulePage ssr />
    </body>
  </html>
));
