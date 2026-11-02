import { derived, writable } from 'svelte/store';
import en from './en.js';
import th from './th.js';

const dictionaries = { en, th };
const DEFAULT_LOCALE = 'th';
const STORAGE_KEY = 'jellyshare_locale';
const SUPPORTED = new Set(['en', 'th']);

function storedLocale() {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (SUPPORTED.has(value)) return value;
  } catch (e) {
    // Storage blocked: use the default.
  }
  return DEFAULT_LOCALE;
}

export const locale = writable(storedLocale());

locale.subscribe((value) => {
  try {
    localStorage.setItem(STORAGE_KEY, value);
  } catch (e) {
    // Storage blocked: the choice lasts for this page only.
  }
  if (typeof document !== 'undefined') {
    document.documentElement.lang = value;
  }
});

export function setLocale(value) {
  if (SUPPORTED.has(value)) locale.set(value);
}

function interpolate(text, params) {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (whole, key) =>
    Object.prototype.hasOwnProperty.call(params, key) ? String(params[key]) : whole
  );
}

function lookup(value, key) {
  const dictionary = dictionaries[value] || {};
  const text = dictionary[key];
  if (typeof text === 'string' && text !== '') return text;
  const fallback = en[key];
  return typeof fallback === 'string' ? fallback : key;
}

export function translate(value, key, params) {
  // Plural keys: `key.one` / `key.other`, chosen with Intl.PluralRules. Thai
  // has a single form, so its dictionary only needs `key.other`.
  if (params && typeof params.count === 'number') {
    const rule = new Intl.PluralRules(value).select(params.count);
    const pluralKey = `${key}.${rule}`;
    const dictionary = dictionaries[value] || {};
    if (dictionary[pluralKey] || en[pluralKey]) {
      return interpolate(lookup(value, pluralKey), params);
    }
  }
  return interpolate(lookup(value, key), params);
}

export const t = derived(locale, ($locale) => (key, params) => translate($locale, key, params));

// The tag for Intl formatters. Thai uses Thai month names with the Gregorian
// calendar, as decided for the share page.
export const localeTag = derived(locale, ($locale) => ($locale === 'th' ? 'th-TH' : 'en-GB'));
