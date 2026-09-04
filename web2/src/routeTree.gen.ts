/* Hand-wired until vite regenerates this file via the TanStack router plugin. */
import { Route as rootRoute } from "./routes/__root";
import { Route as IndexRoute } from "./routes/index";
import { Route as LoginRoute } from "./routes/login";
import { Route as AppRoute } from "./routes/_app";
import { Route as DashboardRoute } from "./routes/_app/dashboard";
import { Route as SplatRoute } from "./routes/_app/$";

const appRoute = AppRoute.update({ id: "/_app", getParentRoute: () => rootRoute });

export const routeTree = rootRoute.addChildren([
  IndexRoute.update({ path: "/", getParentRoute: () => rootRoute }),
  LoginRoute.update({ path: "/login", getParentRoute: () => rootRoute }),
  appRoute.addChildren([
    DashboardRoute.update({ path: "/dashboard", getParentRoute: () => appRoute }),
    SplatRoute.update({ path: "$", getParentRoute: () => appRoute }),
  ]),
]);
