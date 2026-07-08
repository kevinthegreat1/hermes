import React, { useEffect, useState } from 'react';
import { CollapsableSection, ComboboxOption, DateTimePicker, InlineField, MultiCombobox, RadioButtonGroup } from '@grafana/ui';
import { dateTime, DateTime, QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { ChannelRef, KeyRef, MyDataSourceOptions, MyQuery, QueryType, TimeField } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

const QUERY_TYPE_OPTIONS: Array<SelectableValue<QueryType>> = [
  { label: 'Telemetry', value: 'telemetry' },
  { label: 'Events', value: 'events' },
];

const TIME_FIELD_OPTIONS: Array<SelectableValue<TimeField>> = [
  { label: 'Receive Time', value: 'ert' },
  { label: 'On-board Time', value: 'time' },
];

function toOptions(values: string[]): Array<ComboboxOption<string>> {
  return values.map((v) => ({ label: v, value: v }));
}

function keyRefToValue(k: KeyRef): string {
  return JSON.stringify(k);
}

function valueToKeyRef(v: string): KeyRef {
  return JSON.parse(v) as KeyRef;
}

function toKeyOptions(entries: KeyRef[]): Array<ComboboxOption<string>> {
  return entries.map((e) => ({
    label: e.key,
    description: `${e.component}.${e.channel}`,
    value: keyRefToValue(e),
  }));
}

function keyValues(keys: KeyRef[]): string[] {
  return keys.map(keyRefToValue);
}

function channelToKey(ch: ChannelRef): string {
  return JSON.stringify(ch);
}

function keyToChannel(key: string): ChannelRef {
  return JSON.parse(key) as ChannelRef;
}

function toChannelOptions(entries: ChannelRef[]): Array<ComboboxOption<string>> {
  return entries.map((e) => ({
    label: `${e.component}.${e.name}`,
    description: e.component,
    value: channelToKey(e),
  }));
}

function channelValues(channels: ChannelRef[]): string[] {
  return channels.map(channelToKey);
}

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  const queryType = query.queryType ?? 'telemetry';

  // Telemetry state
  const [channelOptions, setChannelOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [sourceOptions, setSourceOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [keyOptions, setKeyOptions] = useState<Array<ComboboxOption<string>>>([]);

  const [channelLoading, setChannelLoading] = useState(false);
  const [sourceLoading, setSourceLoading] = useState(false);
  const [keyLoading, setKeyLoading] = useState(false);

  // Event state
  const [eventSourceOptions, setEventSourceOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [eventSourceLoading, setEventSourceLoading] = useState(false);

  // --- Handlers ---

  const onQueryTypeChange = (value: QueryType) => {
    const updated: MyQuery = {
      ...query,
      queryType: value,
      channels: [],
      keys: [],
      sources: [],
    };
    onChange(updated);
    if (value === 'events') {
      onRunQuery();
    }
  };

  const onChannelChange = (options: Array<ComboboxOption<string>>) => {
    const channels = options.map(({ value }) => keyToChannel(value));
    const updated: MyQuery = { ...query, channels, keys: [], sources: [] };
    onChange(updated);
    if (channels.length) {
      onRunQuery();
    }
  };

  const onSourceChange = (options: Array<ComboboxOption<string>>) => {
    const updated: MyQuery = { ...query, sources: options.map(({ value }) => value) };
    onChange(updated);
    if (queryType === 'telemetry' && updated.channels && updated.channels.length) {
      onRunQuery();
    }
    if (queryType === 'events') {
      onRunQuery();
    }
  };

  const onKeyChange = (options: Array<ComboboxOption<string>>) => {
    const updated: MyQuery = { ...query, keys: options.map(({ value }) => valueToKeyRef(value)) };
    onChange(updated);
    if (updated.channels && updated.channels.length) {
      onRunQuery();
    }
  };

  const onTimeFieldChange = (value: TimeField) => {
    onChange({ ...query, timeField: value });
    onRunQuery();
  };

  const onTimeOverrideFromChange = (date?: DateTime) => {
    onChange({ ...query, timeOverrideFrom: date ? date.toISOString() : undefined });
    onRunQuery();
  };

  const onTimeOverrideToChange = (date?: DateTime) => {
    onChange({ ...query, timeOverrideTo: date ? date.toISOString() : undefined });
    onRunQuery();
  };

  // --- Telemetry data loading ---

  useEffect(() => {
    if (queryType !== 'telemetry') {
      return;
    }
    const loadChannels = async () => {
      setChannelLoading(true);
      datasource
        .getChannels()
        .then((entries) => setChannelOptions(toChannelOptions(entries)))
        .catch(() => setChannelOptions([]))
        .finally(() => setChannelLoading(false));
    };
    loadChannels();
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'telemetry') {
      return;
    }
    const loadSources = async () => {
      setSourceLoading(true);
      datasource
        .getSources()
        .then((values) => setSourceOptions(toOptions(values)))
        .catch(() => setSourceOptions([]))
        .finally(() => setSourceLoading(false));
    };
    loadSources();
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'telemetry' || !query.channels || !query.channels.length) {
      setTimeout(() => setKeyOptions([]), 0);
      return;
    }
    const loadKeys = async () => {
      setKeyLoading(true);
      datasource
        .getKeys(query.channels)
        .then((entries) => setKeyOptions(toKeyOptions(entries)))
        .catch(() => setKeyOptions([]))
        .finally(() => setKeyLoading(false));
    }
    loadKeys();
  }, [datasource, queryType, query.channels]);

  // --- Event data loading ---

  useEffect(() => {
    if (queryType !== 'events') {
      return;
    }
    const loadEventSources = async () => {
      setEventSourceLoading(true);
      datasource
        .getEventSources()
        .then((values) => setEventSourceOptions(toOptions(values)))
        .catch(() => setEventSourceOptions([]))
        .finally(() => setEventSourceLoading(false));
    }
    loadEventSources();
  }, [datasource, queryType]);

  return (
    <>
      <div style={{ marginTop: 8, marginBottom: 8 }}>
        <RadioButtonGroup
          id="query-editor-query-type"
          options={QUERY_TYPE_OPTIONS}
          value={queryType}
          onChange={onQueryTypeChange}
          size="sm"
          fullWidth={true}
        />
      </div>

      {queryType === 'telemetry' && (
        <>
          <InlineField label="Channel" labelWidth={16} tooltip="Telemetry channel name" required>
            <MultiCombobox
              id="query-editor-channel"
              data-testid="query-editor-channel"
              options={channelOptions}
              value={channelValues(query.channels ?? [])}
              onChange={onChannelChange}
              loading={channelLoading}
              placeholder="Select channel"
              width={56}
            />
          </InlineField>
          <InlineField label="Source" labelWidth={16} tooltip="FSW source identifier (optional)">
            <MultiCombobox
              id="query-editor-source"
              data-testid="query-editor-source"
              options={sourceOptions}
              value={query.sources}
              onChange={onSourceChange}
              isClearable
              loading={sourceLoading}
              placeholder="All sources"
              width={56}
            />
          </InlineField>
          {keyOptions.length > 1 && (
            <InlineField label="Key" labelWidth={16} tooltip="Value field path for compound channels">
              <MultiCombobox
                id="query-editor-key"
                data-testid="query-editor-key"
                options={keyOptions}
                value={keyValues(query.keys ?? [])}
                onChange={onKeyChange}
                isClearable
                loading={keyLoading}
                placeholder="All keys"
                width={56}
              />
            </InlineField>
          )}
        </>
      )}

      {queryType === 'events' && (
        <>
          <InlineField label="Source" labelWidth={16} tooltip="FSW source identifier (optional)">
            <MultiCombobox
              id="query-editor-event-source"
              data-testid="query-editor-event-source"
              options={eventSourceOptions}
              value={query.sources}
              onChange={onSourceChange}
              isClearable
              loading={eventSourceLoading}
              placeholder="All sources"
              width={56}
            />
          </InlineField>
        </>
      )}

      <div style={{ marginTop: 8, marginBottom: 8 }}>
        <RadioButtonGroup
          id="query-editor-time-field"
          options={TIME_FIELD_OPTIONS}
          value={query.timeField ?? 'ert'}
          onChange={onTimeFieldChange}
          size="sm"
          fullWidth={false}
        />
      </div>
      <CollapsableSection label="Time override" isOpen={false}>
        <InlineField label="From" labelWidth={16} tooltip="Absolute start time (optional)">
          <DateTimePicker
            date={query.timeOverrideFrom ? dateTime(query.timeOverrideFrom) : undefined}
            onChange={onTimeOverrideFromChange}
            clearable
          />
        </InlineField>
        <InlineField label="To" labelWidth={16} tooltip="Absolute end time (optional)">
          <DateTimePicker
            date={query.timeOverrideTo ? dateTime(query.timeOverrideTo) : undefined}
            onChange={onTimeOverrideToChange}
            clearable
          />
        </InlineField>
      </CollapsableSection>
    </>
  );
}
