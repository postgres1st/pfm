export const isFirstLogin = () => {
  const userId = window.grafanaBootData?.user?.id;
  return localStorage.getItem(`pfm-ui.first-login.user-${userId}`) !== 'false';
};

export const updateIsFirstLogin = () => {
  const userId = window.grafanaBootData?.user?.id;
  localStorage.setItem(`pfm-ui.first-login.user-${userId}`, 'false');
};

export const isUserLoggedIn = () => {
  return window.grafanaBootData?.user?.isSignedIn === true;
};
