import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import ParentTariffsPage from "./screens/ParentTariffsPage.vue";

export const parentTariffsPageRoute = app
  .get("/")
  .query((s) => ({
    date: s.string().optional(),
    child: s.string().optional(),
    tariff: s.string().optional(),
  }))
  .handle((ctx, req) => (
    <html lang="ru">
      <head>
        <meta charset="utf-8" />
        <meta
          name="viewport"
          content="width=device-width, initial-scale=1, viewport-fit=cover"
        />
        <title>Тарифы — мастерская «Ясна»</title>
        <meta name="theme-color" content="#F5EFE4" />
        <meta name="robots" content="noindex, nofollow" />
        <Styles />
      </head>
      <body class="bg-paper">
        <ParentTariffsPage
          ssr
          date={req.query.date ?? ""}
          child={req.query.child ?? ""}
          tariff={req.query.tariff ?? ""}
        />
      </body>
    </html>
  ));
