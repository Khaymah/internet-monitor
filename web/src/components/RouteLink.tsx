import type { ReactNode } from "react";
import { hrefOf, navigate, type Route } from "../lib/router";

export function RouteLink({
  route,
  search = "",
  className,
  children,
  current,
}: {
  route: Route;
  search?: string;
  className?: string;
  children: ReactNode;
  current?: boolean;
}) {
  return (
    <a
      className={className}
      href={hrefOf(route, search)}
      aria-current={current ? "page" : undefined}
      onClick={(e) => {
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
        e.preventDefault();
        navigate(route, search);
      }}
    >
      {children}
    </a>
  );
}
