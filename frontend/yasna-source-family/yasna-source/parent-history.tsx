import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import ParentHistoryPage from "./screens/ParentHistoryPage.vue";

export const parentHistoryPageRoute = app.get("/", () => (
  <html lang="ru">
    <head>
      <meta charset="utf-8" />
      <meta
        name="viewport"
        content="width=device-width, initial-scale=1, viewport-fit=cover"
      />
      <title>История посещений — мастерская «Ясна»</title>
      <meta name="theme-color" content="#F5EFE4" />
      <meta name="robots" content="noindex, nofollow" />
      <Styles />
    </head>
    <body class="bg-paper">
      <ParentHistoryPage ssr />
    </body>
  </html>
));
