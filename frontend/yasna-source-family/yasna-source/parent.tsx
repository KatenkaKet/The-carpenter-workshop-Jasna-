import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import ParentSchedulePage from "./screens/ParentSchedulePage.vue";

export const parentPageRoute = app.get("/", () => (
  <html lang="ru">
    <head>
      <meta charset="utf-8" />
      <meta
        name="viewport"
        content="width=device-width, initial-scale=1, viewport-fit=cover"
      />
      <title>Кабинет родителя — мастерская «Ясна»</title>
      <meta name="theme-color" content="#F5EFE4" />
      <meta name="robots" content="noindex, nofollow" />
      <Styles />
    </head>
    <body class="bg-paper">
      <ParentSchedulePage ssr />
    </body>
  </html>
));
