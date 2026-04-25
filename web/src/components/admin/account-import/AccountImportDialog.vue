<script setup lang="ts">
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

const activePane = ref<ImportPaneKind>('access_token')
const onlyFailed = ref(false)

const advanced = reactive<ImportAdvancedOptionsModel>({
  update_existing: true,
  default_proxy_id: undefined,
  target_pool_id: undefined,
  resolve_identity: true,
  kick_refresh: true,
  kick_quota_probe: true,
})

const accessTokenModel = reactive<AccessTokenImportModel>({
  mode: 'at',
  tokens_text: '',
  client_id: '',
})

const cpaModel = reactive<FileImportModel>({
  text: '',
  files: [],
})

const sub2apiModel = reactive<FileImportModel>({
  text: '',
  files: [],
})

const manualModel = reactive<ManualAccountForm>({
  email: '',
  auth_token: '',
  refresh_token: '',
  session_token: '',
  client_id: '',
  account_type: 'codex',
  plan_type: 'plus',
  daily_image_quota: 100,
  notes: '',
  proxy_id: undefined,
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

function buildPayload(): DialogSubmitPayload {
  switch (activePane.value) {
    case 'cpa':
      return { kind: 'cpa', advanced: { ...advanced }, payload: { ...cpaModel, files: [...cpaModel.files] } }
    case 'sub2api':
      return {
        kind: 'sub2api',
        advanced: { ...advanced },
        payload: { ...sub2apiModel, files: [...sub2apiModel.files] },
      }
    case 'manual':
      return { kind: 'manual', advanced: { ...advanced }, payload: { ...manualModel } }
    default:
      return { kind: 'access_token', advanced: { ...advanced }, payload: { ...accessTokenModel } }
  }
}

function onSubmit() {
  emit('submit', buildPayload())
}
</script>

<template>
  <el-dialog v-model="visible" title="导入账号" width="960px" destroy-on-close>
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
            <ManualAccountPane
              v-model="manualModel"
              :disabled="loading"
              :proxy-options="proxyOptions"
            />
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
