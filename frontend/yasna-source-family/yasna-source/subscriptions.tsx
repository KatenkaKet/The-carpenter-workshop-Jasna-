import { jsx } from "@app/html-jsx";
import { Styles } from "./styles";
import SubscriptionsPage from "./screens/SubscriptionsPage.vue";

export const subscriptionsPageRoute = app
  .get("/")
  .query((s) => ({ tab: s.string().optional() }))
  .handle((ctx, req) => {
    const tab = req.query.tab === "rules" ? "rules" : "list";

    return (
      <html lang="ru">
        <head>
          <meta charset="utf-8" />
          <meta
            name="viewport"
            content="width=device-width, initial-scale=1, viewport-fit=cover"
          />
          <title>Абонементы — мастерская «Ясна»</title>
          <meta name="theme-color" content="#F5EFE4" />
          <meta name="robots" content="noindex, nofollow" />
          <Styles />
        </head>
        <body class="bg-paper">
          <SubscriptionsPage ssr tab={tab} />
        </body>
      </html>
    );
  });
