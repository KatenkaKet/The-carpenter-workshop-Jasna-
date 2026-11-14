import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import ParentDayPage from "./screens/ParentDayPage.vue";
import { DEMO_TODAY } from "./shared/demo";

export const parentDayPageRoute = app
  .get("/")
  .query((s) => ({ date: s.string().optional() }))
  .handle((ctx, req) => {
    const raw = req.query.date ?? "";
    const date = /^\d{4}-\d{2}-\d{2}$/.test(raw) ? raw : DEMO_TODAY;

    return (
      <html lang="ru">
        <head>
          <meta charset="utf-8" />
          <meta
            name="viewport"
            content="width=device-width, initial-scale=1, viewport-fit=cover"
          />
          <title>Запись на занятие — мастерская «Ясна»</title>
          <meta name="theme-color" content="#F5EFE4" />
          <meta name="robots" content="noindex, nofollow" />
          <Styles />
        </head>
        <body class="bg-paper">
          <ParentDayPage ssr date={date} />
        </body>
      </html>
    );
  });
