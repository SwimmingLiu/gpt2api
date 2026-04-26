<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import {
  createAccountImportSource,
  listAccountImportSources,
  listRemoteSub2APIAccounts,
  type AccountImportSource,
  type CreateAccountImportSourceBody,
  type RemoteSub2APIAccount,
} from '@/api/account-import-sources'

import type { FileImportModel } from './types'

const model = defineModel<FileImportModel>({
  required: true,
})

withDefaults(
  defineProps<{
    disabled?: boolean
  }>(),
  {
    disabled: false,
  },
)

const sources = ref<AccountImportSource[]>([])
const remoteAccounts = ref<RemoteSub2APIAccount[]>([])
const loadingSources = ref(false)
const loadingAccounts = ref(false)
const createVisible = ref(false)
const savingSource = ref(false)

const createForm = reactive<CreateAccountImportSourceBody>({
  source_type: 'sub2api',
  name: '',
  base_url: '',
  auth_mode: 'password',
  email: '',
  password: '',
  api_key: '',
  group_id: '',
})

const sub2apiSources = computed(() => sources.value.filter((item) => item.source_type === 'sub2api'))

watch(
  () => model.value.mode,
  (mode) => {
    if (mode === 'local') {
      model.value.source_id = undefined
      model.value.selected_remote_ids = []
      remoteAccounts.value = []
    }
  },
)

function onPickFiles(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files) return
  model.value.files = Array.from(input.files)
  input.value = ''
}

function clearFiles() {
  model.value.files = []
}

async function fetchSources() {
  loadingSources.value = true
  try {
    const result = await listAccountImportSources()
    sources.value = result.items
  } catch {
    sources.value = []
  } finally {
    loadingSources.value = false
  }
}

async function fetchRemoteAccounts() {
  if (!model.value.source_id) {
    ElMessage.warning('请先选择一个 sub2api 远程源')
    return
  }
  loadingAccounts.value = true
  try {
    const result = await listRemoteSub2APIAccounts(model.value.source_id)
    remoteAccounts.value = result.items
    model.value.selected_remote_ids = []
  } finally {
    loadingAccounts.value = false
  }
}

async function createSource() {
  if (!createForm.name?.trim() || !createForm.base_url?.trim()) {
    ElMessage.warning('请填写名称和 base_url')
    return
  }
  if (createForm.auth_mode === 'password') {
    if (!createForm.email?.trim() || !createForm.password?.trim()) {
      ElMessage.warning('Password 模式需要填写 email 和 password')
      return
    }
  }
  if (createForm.auth_mode === 'api_key' && !createForm.api_key?.trim()) {
    ElMessage.warning('API Key 模式需要填写 api_key')
    return
  }
  savingSource.value = true
  try {
    const created = await createAccountImportSource({
      source_type: 'sub2api',
      name: createForm.name.trim(),
      base_url: createForm.base_url.trim(),
      auth_mode: createForm.auth_mode,
      email: createForm.auth_mode === 'password' ? createForm.email?.trim() : undefined,
      password: createForm.auth_mode === 'password' ? createForm.password?.trim() : undefined,
      api_key: createForm.auth_mode === 'api_key' ? createForm.api_key?.trim() : undefined,
      group_id: createForm.group_id?.trim() || undefined,
    })
    createVisible.value = false
    createForm.name = ''
    createForm.base_url = ''
    createForm.email = ''
    createForm.password = ''
    createForm.api_key = ''
    createForm.group_id = ''
    await fetchSources()
    model.value.source_id = created.id
    ElMessage.success(`已创建远程源 ${created.name}`)
  } finally {
    savingSource.value = false
  }
}

function onSelectionChange(rows: RemoteSub2APIAccount[]) {
  model.value.selected_remote_ids = rows.map((row) => row.id)
}

onMounted(() => {
  fetchSources()
})
</script>

<template>
  <div class="pane">
    <el-radio-group v-model="model.mode" :disabled="disabled" style="margin-bottom: 16px">
      <el-radio-button label="local">本地文件</el-radio-button>
      <el-radio-button label="remote">远程 sub2api</el-radio-button>
    </el-radio-group>

    <div v-if="model.mode === 'local'">
      <el-alert
        type="info"
        :closable="false"
        title="支持本地 sub2api JSON 文件，后续由父组件决定如何调用统一导入接口。"
      />

      <el-form label-width="120px" style="margin-top: 16px">
        <el-form-item label="sub2api 文件">
          <div class="file-actions">
            <input accept=".json" :disabled="disabled" type="file" multiple @change="onPickFiles">
            <div class="pane-hint">可上传一个或多个 JSON 文件。</div>
          </div>
        </el-form-item>

        <el-form-item v-if="model.files.length" label="已选文件">
          <div class="file-list">
            <el-tag
              v-for="(file, index) in model.files"
              :key="`${file.name}-${index}`"
              :disable-transitions="true"
            >
              {{ file.name }}
            </el-tag>
            <el-button link type="primary" :disabled="disabled" @click="clearFiles">清空</el-button>
          </div>
        </el-form-item>

        <el-form-item label="文本内容">
          <el-input
            v-model="model.text"
            :autosize="{ minRows: 8, maxRows: 14 }"
            :disabled="disabled"
            type="textarea"
            placeholder="也可以粘贴 sub2api JSON 文本"
          />
        </el-form-item>
      </el-form>
    </div>

    <div v-else>
      <div class="remote-toolbar">
        <el-select
          v-model="model.source_id"
          :disabled="disabled || loadingSources"
          clearable
          filterable
          placeholder="选择一个远程 sub2api 源"
          style="min-width: 280px"
        >
          <el-option
            v-for="item in sub2apiSources"
            :key="item.id"
            :label="item.name"
            :value="item.id"
          />
        </el-select>
        <el-button :disabled="disabled" :loading="loadingSources" @click="fetchSources">刷新源</el-button>
        <el-button :disabled="disabled" @click="createVisible = true">新增源</el-button>
        <el-button type="primary" :disabled="disabled || !model.source_id" :loading="loadingAccounts" @click="fetchRemoteAccounts">
          拉取账号
        </el-button>
      </div>

      <div class="pane-hint" style="margin-bottom: 12px">
        远程模式会从已保存的 sub2api 源拉取账号列表，勾选后导入。
      </div>

      <el-table
        :data="remoteAccounts"
        max-height="320"
        size="small"
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column prop="name" label="名称" min-width="220" />
        <el-table-column prop="email" label="邮箱" min-width="200" />
        <el-table-column prop="plan_type" label="套餐" width="110" />
        <el-table-column prop="status" label="状态" width="110" />
      </el-table>
    </div>

    <el-dialog v-model="createVisible" title="新增 sub2api 远程源" width="560px" append-to-body>
      <el-form label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="createForm.name" placeholder="例如 sub2api 主池" />
        </el-form-item>
        <el-form-item label="Base URL">
          <el-input v-model="createForm.base_url" placeholder="https://example.com" />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-select v-model="createForm.auth_mode" style="width: 100%">
            <el-option label="Email + Password" value="password" />
            <el-option label="API Key" value="api_key" />
          </el-select>
        </el-form-item>
        <template v-if="createForm.auth_mode === 'password'">
          <el-form-item label="Email">
            <el-input v-model="createForm.email" />
          </el-form-item>
          <el-form-item label="Password">
            <el-input v-model="createForm.password" show-password />
          </el-form-item>
        </template>
        <el-form-item v-else label="API Key">
          <el-input v-model="createForm.api_key" show-password />
        </el-form-item>
        <el-form-item label="默认分组">
          <el-input v-model="createForm.group_id" placeholder="可选，导入前浏览账号时会带上 group 参数" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingSource" @click="createSource">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.file-actions {
  width: 100%;
}

.file-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.pane-hint {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.remote-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
</style>
