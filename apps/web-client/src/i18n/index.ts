import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import en from './locales/en.json';
import si from './locales/si.json';
import ta from './locales/ta.json';

export type Locale = 'en' | 'si' | 'ta';

export const supportedLocales: Array<{ code: Locale; native: string; label: string }> = [
  { code: 'en', native: 'English', label: 'English' },
  { code: 'si', native: 'සිංහල', label: 'Sinhala' },
  { code: 'ta', native: 'தமிழ்', label: 'Tamil' },
];

const storedLocale =
  typeof window !== 'undefined'
    ? (localStorage.getItem('cropdoc.locale') as Locale | null)
    : null;

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    si: { translation: si },
    ta: { translation: ta },
  },
  lng: storedLocale ?? 'en',
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
  returnNull: false,
});

export function setLocale(locale: Locale): void {
  void i18n.changeLanguage(locale);
  if (typeof window !== 'undefined') {
    localStorage.setItem('cropdoc.locale', locale);
    document.documentElement.lang = locale;
  }
}

/** Picks the best translation from a schema's multilingual string. */
export function pickLocalised(
  map: Record<string, string> | undefined,
  preferred: Locale,
): string | undefined {
  if (!map) return undefined;
  return map[preferred] ?? map.en ?? Object.values(map)[0];
}

export default i18n;
