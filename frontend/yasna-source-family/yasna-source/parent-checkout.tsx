import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import ParentCheckoutPage from "./screens/ParentCheckoutPage.vue";

export const parentCheckoutPageRoute = app
  .get("/")
  .query((s) => ({
    tariff: s.string().optional(),
    child: s.string().optional(),
    lesson: s.string().optional(),
    date: s.string().optional(),
  }))
  .handle((ctx, req) => (
    <html lang="ru">
      <head>
        <meta charset="utf-8" />
        <meta
          name="viewport"
          content="width=device-width, initial-scale=1, viewport-fit=cover"
        />
        <title>Оплата — мастерская «Ясна»</title>
        <meta name="theme-color" content="#F5EFE4" />
        <meta name="robots" content="noindex, nofollow" />
        <Styles />
      </head>
      <body class="bg-paper">
        <ParentCheckoutPage
          ssr
          tariff={req.query.tariff ?? ""}
          child={req.query.child ?? ""}
          lesson={req.query.lesson ?? ""}
          date={req.query.date ?? ""}
        />
      </body>
    </html>
  ));
