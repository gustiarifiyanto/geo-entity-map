// Translates what the backend (and the zod schemas, which mirror it) say in
// English: value names (types, statuses, …) and validation/error messages.
// The API contract stays English; only the display changes. Anything not
// recognized is shown as it came, so no error is ever hidden.

export type Lang = 'en' | 'id'

/** "iot_device" -> "Iot Device"; the fallback for values no dictionary knows. */
export function titleCase(value: string): string {
  return value
    .split('_')
    .filter(Boolean)
    .map((word) => word[0].toUpperCase() + word.slice(1))
    .join(' ')
}

const VALUE_LABELS: Record<Lang, Record<string, string>> = {
  en: {
    iot_device: 'IoT Device',
    in_progress: 'In progress',
    completed_on_time: 'Completed on time',
    completed_late: 'Completed late',
    water_level: 'Water level',
    wind_speed: 'Wind speed',
  },
  id: {
    vehicle: 'Kendaraan',
    iot_device: 'Perangkat IoT',
    facility: 'Fasilitas',
    active: 'Aktif',
    inactive: 'Tidak aktif',
    maintenance: 'Perawatan',
    user: 'User',
    admin: 'Admin',
    scheduled: 'Terjadwal',
    in_progress: 'Berjalan',
    overdue: 'Terlambat',
    completed_on_time: 'Selesai tepat waktu',
    completed_late: 'Selesai terlambat',
    temperature: 'Suhu',
    water_level: 'Tinggi air',
    wind_speed: 'Kecepatan angin',
  },
}

/**
 * A label for a value sent by the backend (type, status, role, metric,
 * installation status): the dictionary label, else `fallback` (e.g. a label
 * the backend sent along), else the value in title case.
 */
export function valueLabel(lang: Lang, value: string, fallback?: string): string {
  return VALUE_LABELS[lang][value] ?? fallback ?? titleCase(value)
}

// Indonesian versions of the English fragments the backend and the schemas
// produce. Field fragments read after the field label ("Name is required").
type Rule = [RegExp, (...groups: string[]) => string]

const ID_FIELD_RULES: Rule[] = [
  [/^is required$/, () => 'wajib diisi'],
  [/^is invalid$/, () => 'tidak valid'],
  [/^must be a valid value$/, () => 'harus berupa nilai yang valid'],
  [/^must be at most (\d+) characters$/, (n) => `maksimal ${n} karakter`],
  [/^must be at least (\d+) characters$/, (n) => `minimal ${n} karakter`],
  [/^must be at most (\d+) bytes$/, (n) => `maksimal ${n} byte`],
  [/^must be at most (\d+) MB$/, (n) => `maksimal ${n} MB`],
  [/^must be between (\S+) and (\S+)$/, (a, b) => `harus di antara ${a} dan ${b}`],
  [/^must be one of: (.+)$/, (list) => `harus salah satu dari: ${list}`],
  [/^must be a valid email address$/, () => 'harus berupa alamat email yang valid'],
  [/^is already registered$/, () => 'sudah terdaftar'],
  [/^must be a number$/, () => 'harus berupa angka'],
  [/^must be a string$/, () => 'harus berupa teks'],
  [/^must be a boolean$/, () => 'harus berupa true/false'],
  [/^must be a finite number$/, () => 'harus berupa angka yang valid'],
  [/^is out of range$/, () => 'di luar jangkauan'],
  [/^must be valid JSON$/, () => 'harus berupa JSON yang valid'],
  [/^must be a JSON object$/, () => 'harus berupa objek JSON'],
  [/^must be a date \(YYYY-MM-DD\)$/, () => 'harus berupa tanggal (YYYY-MM-DD)'],
  [/^must be a date-time \(RFC3339.*\)$/, () => 'harus berupa tanggal-waktu (RFC3339)'],
  [/^must be on or after the start date$/, () => 'tidak boleh sebelum tanggal mulai'],
  [/^cannot be in the future$/, () => 'tidak boleh di masa depan'],
  [/^cannot be older than (\d+) days$/, (n) => `tidak boleh lebih lama dari ${n} hari`],
  [/^must be a JPEG, PNG or WebP image$/, () => 'harus berupa gambar JPEG, PNG, atau WebP'],
  [/^must be a whole number from (\d+) to (\d+)$/, (a, b) => `harus bilangan bulat ${a} sampai ${b}`],
]

// Whole sentences: backend error messages and messages without a field label.
const ID_SENTENCES: Rule[] = [
  [/^this entity already has the maximum of (\d+) photos$/, (n) => `entitas ini sudah punya maksimal ${n} foto`],
  [/^Name is required$/, () => 'Nama wajib diisi'],
  [/^Name is used more than once$/, () => 'Nama dipakai lebih dari sekali'],
  [/^entity not found$/, () => 'entitas tidak ditemukan'],
  [/^photo not found$/, () => 'foto tidak ditemukan'],
  [/^route not found$/, () => 'alamat tidak ditemukan'],
  [/^login required$/, () => 'perlu masuk terlebih dahulu'],
  [/^admin role required$/, () => 'perlu role admin'],
  [/^invalid email or password$/, () => 'email atau kata sandi salah'],
  [/^invalid device key$/, () => 'key perangkat tidak valid'],
  [/^this entity type has no installation data$/, () => 'tipe entitas ini tidak punya data pemasangan'],
  [/^this entity type has no sensor data$/, () => 'tipe entitas ini tidak punya data sensor'],
  [/^this entity type has no operating zone$/, () => 'tipe entitas ini tidak punya zona operasional'],
  [/^choose the sensor metric first$/, () => 'pilih metrik sensor terlebih dahulu'],
  [/^an unexpected error occurred$/, () => 'terjadi kesalahan yang tidak terduga'],
  [/^method not allowed$/, () => 'metode tidak diizinkan'],
  [/^request body must not be empty$/, () => 'isi permintaan tidak boleh kosong'],
  [/^request body must be a valid JSON object$/, () => 'isi permintaan harus berupa objek JSON yang valid'],
  [/^request body must contain a single JSON object$/, () => 'isi permintaan hanya boleh berisi satu objek JSON'],
  [/^request body must not exceed (\d+) bytes$/, (n) => `isi permintaan tidak boleh lebih dari ${n} byte`],
  [
    /^request body must be multipart\/form-data with a "photo" file$/,
    () => 'isi permintaan harus multipart/form-data dengan file "photo"',
  ],
  [/^only one photo can be uploaded per request$/, () => 'hanya satu foto per permintaan'],
  [/^malformed multipart body$/, () => 'isi multipart rusak'],
  [/^unknown field "(.+)"$/, (field) => `kolom tidak dikenal "${field}"`],
]

function applyRules(rules: Rule[], text: string): string | null {
  for (const [pattern, render] of rules) {
    const match = pattern.exec(text)
    if (match) return render(...match.slice(1))
  }
  return null
}

/** "Name" + "is required" -> "Name is required" / "Nama wajib diisi". */
export function fieldError(lang: Lang, label: string, fragment: string): string {
  if (lang === 'en') return `${label} ${fragment}`
  const translated = applyRules(ID_FIELD_RULES, fragment)
  return translated ? `${label} ${translated}` : `${label} ${fragment}`
}

/** A message that stands on its own (an API error message, a row error). */
export function sentence(lang: Lang, text: string): string {
  if (lang === 'en') return text
  return applyRules(ID_SENTENCES, text) ?? text
}
