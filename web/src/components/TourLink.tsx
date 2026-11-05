import { CirclePlay } from "lucide-react";
import type { TourId } from "../content/tours";
import { useI18n } from "../lib/i18n";
import { RouteLink } from "./RouteLink";

export function TourLink({ id }: { id: TourId }) {
  const { t } = useI18n();
  return (
    <RouteLink route="tutorial" search={`?v=${id}`} className="btn btn-ghost btn-sm tour-link no-print">
      <CirclePlay size={16} aria-hidden />
      {t("tour_how")}
    </RouteLink>
  );
}
