<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import AccessTokenImportPane from './AccessTokenImportPane.vue'
import CPAImportPane from './CPAImportPane.vue'
import ImportAdvancedOptions from './ImportAdvancedOptions.vue'
import ManualAccountPane from './ManualAccountPane.vue'
import Sub2APIImportPane from './Sub2APIImportPane.vue'
import type {
  AccessTokenImportModel,
  DialogSubmitPayload,
  FileImportModel,
  ImportAdvancedOptions as ImportAdvancedOptionsModel,
  ImportAdvancedOptionsVisibility,
  ImportDialogResultRow,
  ImportPaneKind,
  ManualAccountForm,
  SelectOption,
} from './types'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean
    loading?: boolean
    resultRows?: ImportDialogResultRow[]
    proxyOptions?: SelectOption[]
    poolOptions?: SelectOption[]
  }>(),
  {
    modelValue: false,
    loading: false,
    resultRows: () => [],
    proxyOptions: () => [],
    poolOptions: () => [],
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [payload: DialogSubmitPayload]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const defaultAdvancedState = (): ImportAdvancedOptionsModel => ({
  update_existing: true,
  default_proxy_id: undefined,
  target_pool_id: undefined,
  resolve_identity: true,
  kick_refresh: true,
  kick_quota_probe: true,
})

const defaultAccessTokenModel = (): AccessTokenImportModel => ({
  mode: 'at',
  tokens_text: '',
  client_id: '',
})

const defaultFileImportModel = (): FileImportModel => ({
  mode: 'local',
  text: '',
  files: [],
  source_id: undefined,
  selected_remote_ids: [],
})

const defaultManualModel = (): ManualAccountForm => ({
  email: '',
  auth_token: '',
  refresh_token: '',
  session_token: '',
  client_id: '',
  account_type: 'codex',
  plan_type: 'plus',
  daily_image_quota: 100,
  notes: '',
})

const activePane = ref<ImportPaneKind>('access_token')
const onlyFailed = ref(false)

const advanced = reactive<ImportAdvancedOptionsModel>(defaultAdvancedState())

const accessTokenModel = reactive<AccessTokenImportModel>(defaultAccessTokenModel())

const cpaModel = reactive<FileImportModel>(defaultFileImportModel())

const sub2apiModel = reactive<FileImportModel>(defaultFileImportModel())

const manualModel = reactive<ManualAccountForm>(defaultManualModel())

const advancedVisibility = computed<ImportAdvancedOptionsVisibility>(() => {
  if (activePane.value === 'manual') {
    return {
      show_update_existing: false,
    }
  }
  return {}
})

const filteredRows = computed(() => {
  if (!onlyFailed.value) return props.resultRows
  return props.resultRows.filter((row) => row.status === 'failed' || row.status === 'skipped')
})

watch(
  () => props.modelValue,
  (value) => {
    if (!value) {
      onlyFailed.value = false
    }
  },
)

function assertNeverPane(value: never): never {
  throw new Error(`unsupported import pane: ${String(value)}`)
}

function buildPayload(): DialogSubmitPayload {
  switch (activePane.value) {
    case 'access_token':
      return { kind: 'access_token', advanced: { ...advanced }, payload: { ...accessTokenModel } }
    case 'cpa':
      return { kind: 'cpa', advanced: { ...advanced }, payload: { ...cpaModel, files: [...cpaModel.files] } }
    case 'sub2api':
      return {
        kind: 'sub2api',
        advanced: { ...advanced },
        payload: { ...sub2apiModel, files: [...sub2apiModel.files] },
      }
    case 'manual':
      return {
        kind: 'manual',
        advanced: { ...advanced, update_existing: false },
        payload: { ...manualModel },
      }
    default:
      return assertNeverPane(activePane.value)
  }
}

function normalizedTokenLines(text: string) {
  return text
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function resetDialogState() {
  activePane.value = 'access_token'
  onlyFailed.value = false
  Object.assign(advanced, defaultAdvancedState())
  Object.assign(accessTokenModel, defaultAccessTokenModel())
  Object.assign(cpaModel, defaultFileImportModel())
  Object.assign(sub2apiModel, defaultFileImportModel())
  Object.assign(manualModel, defaultManualModel())
}

function validateBeforeSubmit() {
  if (activePane.value === 'access_token') {
    const tokens = normalizedTokenLines(accessTokenModel.tokens_text)
    if (tokens.length === 0) {
      ElMessage.warning('请至少提供一条 token')
      return false
    }
    if (accessTokenModel.mode === 'rt' && !accessTokenModel.client_id.trim()) {
      ElMessage.warning('RT 模式必须填写 client_id')
      return false
    }
    return true
  }

  if (activePane.value === 'manual') {
    if (!manualModel.email.trim()) {
      ElMessage.warning('手动新增必须填写邮箱')
      return false
    }
    if (!manualModel.auth_token.trim() && !manualModel.refresh_token.trim() && !manualModel.session_token.trim()) {
      ElMessage.warning('手动新增至少需要提供 access_token、refresh_token、session_token 其中之一')
      return false
    }
    return true
  }

  if (activePane.value === 'cpa') {
    if (cpaModel.mode === 'remote') {
      if (!cpaModel.source_id) {
        ElMessage.warning('请选择一个 CPA 远程源')
        return false
      }
      if (cpaModel.selected_remote_ids.length === 0) {
        ElMessage.warning('请至少选择一个远程 CPA 文件')
        return false
      }
      return true
    }
    if (!cpaModel.text.trim() && cpaModel.files.length === 0) {
      ElMessage.warning('请提供 CPA 文本或至少选择一个文件')
      return false
    }
    return true
  }

  if (activePane.value === 'sub2api') {
    if (sub2apiModel.mode === 'remote') {
      if (!sub2apiModel.source_id) {
        ElMessage.warning('请选择一个 sub2api 远程源')
        return false
      }
      if (sub2apiModel.selected_remote_ids.length === 0) {
        ElMessage.warning('请至少选择一个远程 sub2api 账号')
        return false
      }
      return true
    }
    if (!sub2apiModel.text.trim() && sub2apiModel.files.length === 0) {
      ElMessage.warning('请提供 sub2api 文本或至少选择一个文件')
      return false
    }
    return true
  }

  return true
}

function onSubmit() {
  if (!validateBeforeSubmit()) return
  emit('submit', buildPayload())
}

function onDialogClosed() {
  resetDialogState()
}
</script>

<template>
  <el-dialog v-model="visible" title="导入账号" width="960px" destroy-on-close @closed="onDialogClosed">
    <div class="dialog-layout">
      <div class="dialog-main">
        <el-tabs v-model="activePane">
          <el-tab-pane label="Access Token" name="access_token">
            <AccessTokenImportPane v-model="accessTokenModel" :disabled="loading" />
          </el-tab-pane>
          <el-tab-pane label="CPA 文件" name="cpa">
            <CPAImportPane v-model="cpaModel" :disabled="loading" />
          </el-tab-pane>
          <el-tab-pane label="sub2api" name="sub2api">
            <Sub2APIImportPane v-model="sub2apiModel" :disabled="loading" />
          </el-tab-pane>
          <el-tab-pane label="手动新增" name="manual">
            <ManualAccountPane v-model="manualModel" :disabled="loading" />
          </el-tab-pane>
        </el-tabs>
      </div>

      <div class="dialog-side">
        <el-card shadow="never">
          <template #header>
            <span>高级选项</span>
          </template>
          <ImportAdvancedOptions
            v-model="advanced"
            :disabled="loading"
            :pool-options="poolOptions"
            :proxy-options="proxyOptions"
            :visibility="advancedVisibility"
          />
        </el-card>
      </div>
    </div>

    <el-card v-if="resultRows.length" class="result-card" shadow="never">
      <template #header>
        <div class="result-header">
          <span>导入结果</span>
          <el-checkbox v-model="onlyFailed">仅查看失败/跳过</el-checkbox>
        </div>
      </template>

      <el-table :data="filteredRows" max-height="260" size="small">
        <el-table-column label="#" prop="index" width="72" />
        <el-table-column label="来源" min-width="120">
          <template #default="{ row }">
            <span>{{ row.source_type || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="邮箱" prop="email" min-width="180" />
        <el-table-column label="来源引用" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.source_ref || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="110">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'failed' ? 'danger' : row.status === 'skipped' ? 'warning' : 'success'"
              size="small"
            >
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="警告" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.warnings?.join('；') || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="原因" prop="reason" min-width="220" show-overflow-tooltip />
      </el-table>
    </el-card>

    <template #footer>
      <div class="dialog-footer">
        <el-button :disabled="loading" @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="onSubmit">开始导入</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.dialog-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 16px;
  align-items: start;
}

.dialog-main,
.dialog-side {
  min-width: 0;
}

.result-card {
  margin-top: 16px;
}

.result-header,
.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

@media (max-width: 900px) {
  .dialog-layout {
    grid-template-columns: 1fr;
  }
}
</style>
