export const isFirstLogin = () => {
  const userId = window.grafanaBootData?.user?.id;
  return localStorage.getItem(`pfw-ui.first-login.user-${userId}`) !== 'false';
};

export const updateIsFirstLogin = () => {
  const userId = window.grafanaBootData?.user?.id;
  localStorage.setItem(`pfw-ui.first-login.user-${userId}`, 'false');
};

export const isUserLoggedIn = () => {
  return window.grafanaBootData?.user?.isSignedIn === true;
};
