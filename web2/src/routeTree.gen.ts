/* Minimal route tree so `npm run dev` boots before the TanStack plugin regenerates this file. */
import { Route as rootRoute } from "./routes/__root";
import { Route as IndexRoute } from "./routes/index";
import { Route as LoginRoute } from "./routes/login";
import { Route as AppRoute } from "./routes/_app";
import { Route as DashboardRoute } from "./routes/_app/dashboard";
import { Route as TanksRoute } from "./routes/_app/stock.tanks";
import { Route as ReceiptsRoute } from "./routes/_app/stock.receipts";
import { Route as LoadingsRoute } from "./routes/_app/gantry.loadings";
import { Route as PumpoverRoute } from "./routes/_app/terminal.pumpover";
import { Route as BillingRoute } from "./routes/_app/billing";
import { Route as EwuraRoute } from "./routes/_app/ewura";
import { Route as UsersRoute } from "./routes/_app/access.users";
import { Route as RolesRoute } from "./routes/_app/access.roles";

const appRoute = AppRoute.update({ id: "/_app", getParentRoute: () => rootRoute });

export const routeTree = rootRoute.addChildren([
  IndexRoute.update({ path: "/", getParentRoute: () => rootRoute }),
  LoginRoute.update({ path: "/login", getParentRoute: () => rootRoute }),
  appRoute.addChildren([
    DashboardRoute.update({ path: "/dashboard", getParentRoute: () => appRoute }),
    TanksRoute.update({ path: "/stock/tanks", getParentRoute: () => appRoute }),
    ReceiptsRoute.update({ path: "/stock/receipts", getParentRoute: () => appRoute }),
    LoadingsRoute.update({ path: "/gantry/loadings", getParentRoute: () => appRoute }),
    PumpoverRoute.update({ path: "/terminal/pumpover", getParentRoute: () => appRoute }),
    BillingRoute.update({ path: "/billing", getParentRoute: () => appRoute }),
    EwuraRoute.update({ path: "/ewura", getParentRoute: () => appRoute }),
    UsersRoute.update({ path: "/access/users", getParentRoute: () => appRoute }),
    RolesRoute.update({ path: "/access/roles", getParentRoute: () => appRoute }),
  ]),
]);
