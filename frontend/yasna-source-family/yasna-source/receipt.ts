import { createSharedDownloadUrl, createSharedOriginalUrl } from "@app/storage";

export const receiptFileRoute = app
  .get("/")
  .query((s) => ({
    hash: s.string(),
    download: s.string().optional(),
  }))
  .handle(async (ctx, req) => {
    const hash = req.query.hash.trim();
    if (!/^[A-Za-z0-9_.-]+$/.test(hash)) {
      return ctx.resp.redirect("/yasna/finance");
    }

    try {
      const url =
        req.query.download === "1"
          ? await createSharedDownloadUrl(ctx, hash, { expiresInSeconds: 300 })
          : await createSharedOriginalUrl(ctx, hash, { expiresInSeconds: 300 });
      return ctx.resp.redirect(url);
    } catch {
      return ctx.resp.redirect("/yasna/finance?receipt=missing");
    }
  });
