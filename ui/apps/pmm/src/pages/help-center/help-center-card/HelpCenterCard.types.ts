export interface HelpCardButton {
  text: string;
  target?: string;
  url?: string;
  to?: string;
  startIconName?: string;
  onClick?: () => void;
  dataTestId?: string;
}

export interface HelpCard {
  id: string;
  title: string;
  description: string;
  buttons: HelpCardButton[];
  adminOnly: boolean;
  // When true, the card gets the brand amber top-accent (see HelpCenterCard).
  accented?: boolean;
}

export interface HelpCenterCardProps {
  card: HelpCard;
}
